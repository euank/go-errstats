package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPercent(t *testing.T) {
	tests := []struct {
		name     string
		lhs, rhs int64
		want     float64
	}{
		{name: "zero denominator", lhs: 1, rhs: 0, want: 0},
		{name: "zero numerator", lhs: 0, rhs: 10, want: 0},
		{name: "ratio", lhs: 1, rhs: 4, want: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percent(tt.lhs, tt.rhs); got != tt.want {
				t.Fatalf("percent(%d, %d) = %v; want %v", tt.lhs, tt.rhs, got, tt.want)
			}
		})
	}
}

func TestPrettyPrintIncludesDoubleNilWarning(t *testing.T) {
	v := &errStatVisitor{
		lineCount:       10,
		nilNilCount:     1,
		exprLinesMap:    map[string]struct{}{"source.go:1": {}},
		expressionCount: 2,
	}
	var output bytes.Buffer
	v.PrettyPrint(&output)

	for _, want := range []string{"Total lines: \t10", "Number of 'nil != nil'"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestVisitorFindsErrorCheckInCompoundCondition(t *testing.T) {
	source := `package p
func returnsError() error { return nil }
func f() {
	err := returnsError()
	ok := true
	if ok && (err != nil) {}
}`
	v := visitorForSource(t, source)

	if v.conditionCount != 1 || v.errNotNilCount != 1 || v.errNotNilNamedErrCount != 1 {
		t.Fatalf("counts = conditions:%d error-checks:%d named-err:%d; want 1, 1, 1",
			v.conditionCount, v.errNotNilCount, v.errNotNilNamedErrCount)
	}
}

func TestLoadPackagesRejectsInvalidPattern(t *testing.T) {
	if _, err := loadPackages([]string{"./definitely-not-a-package"}, false); err == nil {
		t.Fatal("loadPackages returned nil error for an invalid package pattern")
	}
}

func TestCollectPackages(t *testing.T) {
	leaf := &packages.Package{ID: "leaf"}
	dependency := &packages.Package{ID: "dependency", Imports: map[string]*packages.Package{"leaf": leaf}}
	root := &packages.Package{ID: "root", Imports: map[string]*packages.Package{"dependency": dependency}}

	withoutDependencies := collectPackages([]*packages.Package{root}, false)
	if got := packageIDs(withoutDependencies); !slices.Equal(got, []string{"root"}) {
		t.Fatalf("without dependencies = %v; want [root]", got)
	}

	withDependencies := collectPackages([]*packages.Package{root}, true)
	if got := packageIDs(withDependencies); !slices.Equal(got, []string{"dependency", "leaf", "root"}) {
		t.Fatalf("with dependencies = %v; want [dependency leaf root]", got)
	}
}

func visitorForSource(t *testing.T, source string) *errStatVisitor {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
	if _, err := (&types.Config{}).Check("test/package", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	v := &errStatVisitor{
		pkgInfo:      &packages.Package{TypesInfo: info},
		fset:         fset,
		exprLinesMap: make(map[string]struct{}),
	}
	ast.Walk(v, file)
	return v
}

func packageIDs(pkgs []*packages.Package) []string {
	ids := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		ids[i] = pkg.ID
	}
	return ids
}
