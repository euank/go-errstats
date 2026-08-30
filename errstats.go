package main

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"

	"golang.org/x/tools/go/packages"
)

// ty https://github.com/kisielk/errcheck/blob/master/internal/errcheck/errcheck.go
var errorType *types.Interface

func init() {
	errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
}

type errStatVisitor struct {
	lineCount              int64
	expressionCount        int64
	conditionCount         int64
	errNotNilCount         int64
	errNotNilNamedErrCount int64
	nilNilCount            int64

	verbose      bool
	pkgInfo      *packages.Package
	fset         *token.FileSet
	exprLinesMap map[string]struct{}
}

func percent(lhs, rhs int64) float64 {
	if rhs == 0 {
		return 0
	}
	return float64(lhs) / float64(rhs) * 100.0
}

func (e *errStatVisitor) PrettyPrint(w io.Writer) {
	output := fmt.Sprintf("Statistics about your go files:\n"+
		"\tTotal lines: \t%v\n"+
		"\tTotal meaningful lines: \t%v\n"+
		"\tTotal expressions: \t%v\n"+
		"\tTotal conditionals: \t%v\n"+
		"\tTotal conditionals that were error checks: \t%v\n"+
		"\n"+
		"\tPercent lines that were errchecks: \t%v\n"+
		"\tPercent expressions that were errchecks: \t%v\n"+
		"\tPercent conditionals that were errchecks: \t%v\n"+
		"\tPercent of err != nil checks using the var 'err': \t%v\n",
		e.lineCount,
		len(e.exprLinesMap),
		e.expressionCount,
		e.conditionCount,
		e.errNotNilCount,
		percent(e.errNotNilCount, int64(len(e.exprLinesMap))),
		percent(e.errNotNilCount, e.expressionCount),
		percent(e.errNotNilCount, e.conditionCount),
		percent(e.errNotNilNamedErrCount, e.errNotNilCount),
	)

	if e.nilNilCount > 0 {
		output += fmt.Sprintf("\tNumber of 'nil != nil' and 'nil == nil' conditionals. FIX THIS: \t%v\n", e.nilNilCount)
	}

	fmt.Fprint(w, output)
}

func (e *errStatVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	pos := e.fset.Position(node.Pos())
	line := pos.Filename + ":" + strconv.Itoa(pos.Line)
	e.exprLinesMap[line] = struct{}{}

	handleCond := func(cond ast.Expr) {
		slog.Debug("line: " + line)
		var logbuf bytes.Buffer
		printer.Fprint(&logbuf, e.fset, node)
		slog.Debug("node: " + logbuf.String())

		foundErrCheck := false
		foundNamedErrCheck := false
		var inspectCond func(ast.Expr)
		inspectCond = func(expr ast.Expr) {
			switch expr := expr.(type) {
			case *ast.ParenExpr:
				inspectCond(expr.X)
				return
			case *ast.BinaryExpr:
				if expr.Op == token.LAND || expr.Op == token.LOR {
					inspectCond(expr.X)
					inspectCond(expr.Y)
					return
				}

				if _, isIdent := expr.X.(*ast.Ident); !isIdent {
					// e.g. ast.SelectorExpr
					slog.Debug("skipping, lhs isn't good for us")
					return
				}
				if _, isIdent := expr.Y.(*ast.Ident); !isIdent {
					slog.Debug("skipping, rhs isn't good for us")
					return
				}

				lhs := expr.X.(*ast.Ident)
				rhs := expr.Y.(*ast.Ident)
				if lhs.Name == "nil" && rhs.Name == "nil" {
					slog.Warn("line " + line + " has a double nil check")
					e.nilNilCount++
					return
				}

				if expr.Op != token.NEQ {
					return
				}

				var errIdent *ast.Ident
				if lhs.Name == "nil" {
					errIdent = rhs
				} else if rhs.Name == "nil" {
					errIdent = lhs
				} else {
					return
				}

				thisErr := e.pkgInfo.TypesInfo.Uses[errIdent]
				if thisErr == nil || thisErr.Type() == nil {
					slog.Debug("could not find type for identifier", "identifier", errIdent)
					return
				}
				if types.Implements(thisErr.Type(), errorType) {
					slog.Debug("identified error type")
					foundErrCheck = true

					if errIdent.Name == "err" {
						slog.Debug("identified 'err' name")
						foundNamedErrCheck = true
					}
				}
			}
		}
		inspectCond(cond)
		if foundErrCheck {
			e.errNotNilCount++
		}
		if foundNamedErrCheck {
			e.errNotNilNamedErrCount++
		}
	}

	e.expressionCount++
	switch stmnt := node.(type) {
	case *ast.ForStmt:
		slog.Debug("for statement")
		if stmnt.Cond != nil {
			e.conditionCount++
			handleCond(stmnt.Cond)
		}
	case *ast.IfStmt:
		slog.Debug("if statement")
		e.conditionCount++
		handleCond(stmnt.Cond)
	}

	return e
}

func main() {
	if err := main_(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func main_() error {
	loglevel := flag.String("loglevel", "warn", "loglevel")
	allFiles := flag.Bool("all", false, "stats for all dependencies")
	flag.Parse()

	var lvl slog.Level
	err := lvl.UnmarshalText([]byte(*loglevel))
	if err != nil {
		slog.Error("invalid log level", "err", err)
		os.Exit(1)
	}
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(l)

	pkgs, err := loadPackages(flag.Args(), *allFiles)
	if err != nil {
		return err
	}

	v := &errStatVisitor{
		exprLinesMap: make(map[string]struct{}),
	}

	for _, pkg := range pkgs {
		slog.Info("parsing package", "pkg", pkg.PkgPath)
		v.pkgInfo = pkg
		v.fset = pkg.Fset
		for _, astFile := range pkg.Syntax {
			slog.Debug("parsing", "file", v.fset.Position(astFile.Pos()).Filename, "package", pkg.String())
			v.lineCount += int64(v.fset.File(astFile.Pos()).LineCount())
			ast.Walk(v, astFile)
		}
	}

	v.PrettyPrint(os.Stdout)
	return nil
}

func loadPackages(patterns []string, allFiles bool) ([]*packages.Package, error) {
	loadcfg := &packages.Config{Mode: packages.LoadSyntax}
	if allFiles {
		loadcfg.Mode = packages.LoadAllSyntax
	}
	pkgs, err := packages.Load(loadcfg, patterns...)
	if err != nil {
		return nil, err
	}
	pkgs = collectPackages(pkgs, allFiles)
	var loadErrors []error
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			loadErrors = append(loadErrors, errors.New(pkgErr.Error()))
		}
	}
	if err := errors.Join(loadErrors...); err != nil {
		return nil, err
	}

	return pkgs, nil
}

func collectPackages(pkgs []*packages.Package, allFiles bool) []*packages.Package {
	pkgSet := make(map[string]*packages.Package, len(pkgs))
	toWalk := pkgs
	for len(toWalk) > 0 {
		pkg := toWalk[0]
		toWalk = toWalk[1:]
		if _, ok := pkgSet[pkg.ID]; ok {
			// don't recurse on this one again, we've seen it
			continue
		}
		if allFiles {
			toWalk = slices.AppendSeq(toWalk, maps.Values(pkg.Imports))
		}
		pkgSet[pkg.ID] = pkg
	}

	pkgs = slices.Collect(maps.Values(pkgSet))
	slices.SortFunc(pkgs, func(l, r *packages.Package) int {
		return cmp.Compare(l.ID, r.ID)
	})
	return pkgs
}
