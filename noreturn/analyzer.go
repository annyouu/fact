// returnしない関数に対して、NoReturnFactを付けて
// 別の関数内でその情報を読み取り、使うという例
package noreturn

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Fact定義
type NoReturnFact struct{}

func (*NoReturnFact) AFact() {} // マーカー

// Analyzer定義
var Analyzer = &analysis.Analyzer{
	Name: "noreturn",
	Doc: "return文を持たない関数の呼び出しを検出する",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// 全てのFuncDeclをスキャンし、return文が一度もない関数をFactに登録する
	funcFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(funcFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}

		// 関数内にreturnがあるかどうか調べる
		hasReturn := false
		ast.Inspect(fn.Body, func(n2 ast.Node) bool {
			if _, ok := n2.(*ast.ReturnStmt); ok {
				hasReturn = true
				return false // もう探さなくていい
			}
			return true
		})

		if !hasReturn {
			// returnがひとつもないなら、この関数はreturnしないとみなす
			obj := pass.TypesInfo.ObjectOf(fn.Name)
			pass.ExportObjectFact(obj, &NoReturnFact{})
		}
	})

	// 再度すべての FuncDecl をスキャンし、呼び出し箇所を検出
	insp.Preorder(funcFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}

		// 本体を再帰的に探索し、CallExprを全て調べる
		ast.Inspect(fn.Body, func(n2 ast.Node) bool {
			call, ok := n2.(*ast.CallExpr)
			if !ok {
				return true
			}
			// 呼び出し先が識別子(関数)の場合のみ
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}

			obj := pass.TypesInfo.ObjectOf(ident)
			var fact NoReturnFact
			if pass.ImportObjectFact(obj, &fact) {
				// 事実として「returnしない関数」として登録されていればレポート
				pass.Reportf(call.Lparen, "呼び出された関数 %s は return 文を持ちません", ident.Name)
			}
			return true
		})
	})
	return nil, nil
}
