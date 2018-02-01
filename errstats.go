package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/printer"
	"go/token"
	"go/types"
	"strconv"

	"github.com/Sirupsen/logrus"
	"golang.org/x/tools/go/loader"
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
	pkgInfo      *loader.PackageInfo
	fset         *token.FileSet
	exprLinesMap map[string]struct{}
}

func percent(lhs, rhs int64) float64 {
	if rhs == 0 {
		return 0
	}
	return float64(lhs) / float64(rhs) * 100.0
}

func (e *errStatVisitor) PrettyPrint() {
	output :=
		fmt.Sprintf("Statistics about your go files:\n"+
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
		output = fmt.Sprintf("\tNumber of 'nil != nil' and 'nil == nil' conditionals. FIX THIS: \t%v\n", e.nilNilCount)
	}

	fmt.Printf(output)

}

func (e *errStatVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	pos := e.fset.Position(node.Pos())
	line := pos.Filename + ":" + strconv.Itoa(pos.Line)
	e.exprLinesMap[line] = struct{}{}

	handleCond := func(cond ast.Expr) {
		logrus.Debug("line: " + line)
		var logbuf bytes.Buffer
		printer.Fprint(&logbuf, e.fset, node)
		logrus.Debug("node: " + logbuf.String())

		if neq, ok := cond.(*ast.BinaryExpr); ok {
			if _, isIdent := neq.X.(*ast.Ident); !isIdent {
				// e.g. ast.SelectorExpr
				logrus.Debug("skipping, lhs isn't good for us")
				return
			}
			if _, isIdent := neq.Y.(*ast.Ident); !isIdent {
				logrus.Debug("skipping, rhs isn't good for us")
				return
			}

			if neq.X.(*ast.Ident).Name == "nil" &&
				neq.Y.(*ast.Ident).Name == "nil" {
				logrus.Warn("line " + line + " has a double nil check")
				e.nilNilCount++
				return
			}

			if neq.Op == token.NEQ {
				var el *ast.Ident
				if neq.X.(*ast.Ident).Name == "nil" {
					el = neq.Y.(*ast.Ident)
				} else if neq.Y.(*ast.Ident).Name == "nil" {
					el = neq.X.(*ast.Ident)
				} else {
					// Neither half nil, whatever
					return
				}

				thisErr := e.pkgInfo.Types[el]
				if types.Implements(thisErr.Type, errorType) {
					logrus.Debug("identified 'err' type")
					e.errNotNilCount++

					if el.Name == "err" {
						logrus.Debug("identified 'err' name")
						e.errNotNilNamedErrCount++
					}
				}
			}
		}
	}

	e.expressionCount++
	switch stmnt := node.(type) {
	case *ast.ForStmt:
		logrus.Debug("for statement")
		if stmnt.Cond != nil {
			e.conditionCount++
			handleCond(stmnt.Cond)
		}
	case *ast.IfStmt:
		logrus.Debug("if statement")
		e.conditionCount++
		handleCond(stmnt.Cond)
	}

	return e
}

func main() {
	loglevel := flag.String("loglevel", "info", "loglevel")
	allFiles := flag.Bool("all", false, "stats for all dependencies")
	flag.Parse()

	logLvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		logrus.Fatalf("Cannot parse level: %v", loglevel)
	}
	logrus.SetLevel(logLvl)

	loadcfg := loader.Config{Build: &build.Default}
	loadcfg.FromArgs(flag.Args(), false)

	program, err := loadcfg.Load()
	if err != nil {
		logrus.Fatalf("error loading stuff %v\n", err)
	}

	v := &errStatVisitor{
		fset:         program.Fset,
		exprLinesMap: make(map[string]struct{}),
	}

	var pkgs []*loader.PackageInfo
	if *allFiles {
		pkgs = pkgSlice(program.AllPackages)
	} else {
		pkgs = program.InitialPackages()
	}

	for _, pkg := range pkgs {
		v.pkgInfo = pkg
		for _, astFile := range pkg.Files {
			logrus.WithFields(logrus.Fields{
				"file":    v.fset.Position(astFile.Pos()).Filename,
				"package": pkg.String(),
			}).Debug()

			v.lineCount += int64(v.fset.File(astFile.Pos()).LineCount())
			ast.Walk(v, astFile)
		}
	}

	v.PrettyPrint()
}

func pkgSlice(m map[*types.Package]*loader.PackageInfo) []*loader.PackageInfo {
	output := make([]*loader.PackageInfo, len(m))
	i := 0
	for _, el := range m {
		output[i] = el
		i++
	}
	return output
}
