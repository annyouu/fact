package paniccheck

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type isPanicker struct{}

func (*isPanicker) AFact() {}

var Analyzer = &analysis.Analyzer{
	Name:      "paniccheck",
	Doc:       "panic を呼び出し、recover がない関数を検出する",
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{&isPanicker{}},
	Run:       run,
}

type funcInfo struct {
	hasPanic   bool
	hasRecover bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
		(*ast.DeferStmt)(nil),
	}

	fnInfo := make(map[*types.Func]*funcInfo)
	var currentFunc *types.Func

	// panic recover の検出
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if fn, ok := pass.TypesInfo.ObjectOf(n.Name).(*types.Func); ok {
				currentFunc = fn
				fnInfo[fn] = &funcInfo{}
			}
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok {
				if obj := pass.TypesInfo.ObjectOf(id); obj != nil {
					if builtin, ok := obj.(*types.Builtin); ok {
						switch builtin.Name() {
						case "panic":
							if currentFunc != nil {
								fnInfo[currentFunc].hasPanic = true
							}
						case "recover":
							if currentFunc != nil {
								fnInfo[currentFunc].hasRecover = true
							}
						}
					}
				}
			}
		case *ast.DeferStmt:
			call := n.Call
			if id, ok := call.Fun.(*ast.Ident); ok {
				if obj := pass.TypesInfo.ObjectOf(id); obj != nil {
					if builtin, ok := obj.(*types.Builtin); ok && builtin.Name() == "recover" {
						if currentFunc != nil {
							fnInfo[currentFunc].hasRecover = true
						}
					}
				}
			}
		}
	})

	// panicの伝搬分析
	changed := true
	for changed {
		changed = false
		currentFunc = nil
		inspect.Preorder(nodeFilter, func(n ast.Node) {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if fn, ok := pass.TypesInfo.ObjectOf(n.Name).(*types.Func); ok {
					currentFunc = fn
				}
			case *ast.CallExpr:
				var calledFunc *types.Func
				switch fun := n.Fun.(type) {
				case *ast.Ident:
					if fn, ok := pass.TypesInfo.ObjectOf(fun).(*types.Func); ok {
						calledFunc = fn
					}
				case *ast.SelectorExpr:
					if sel := pass.TypesInfo.Selections[fun]; sel != nil {
						if fn, ok := sel.Obj().(*types.Func); ok {
							calledFunc = fn
						}
					} else {
						// 外部パッケージなどの関数呼び出し
						if fn, ok := pass.TypesInfo.ObjectOf(fun.Sel).(*types.Func); ok {
							calledFunc = fn
						}
					}
				}

				if currentFunc != nil && calledFunc != nil {
					// 呼び出された関数が panic を含み、かつ自分はまだ含んでいないなら伝搬
					if fnInfo[calledFunc].hasPanic && !fnInfo[currentFunc].hasPanic {
						fnInfo[currentFunc].hasPanic = true
						changed = true
					}
				}
			}
		})
	}

	// 検出報告
	for fn, info := range fnInfo {
		if info.hasPanic && !info.hasRecover {
			pass.ExportObjectFact(fn, &isPanicker{})
			pass.Reportf(fn.Pos(), "関数 %s が panic を呼び出しますが recover がありません", fn.Name())
		}
	}

	return nil, nil
}
