package ast

type Program struct {
	PackageName string
	Functions   []FuncDecl
}

type FuncDecl struct {
	Name       string
	Params     []Param
	ReturnType Type
	Body       Block
}

type Param struct {
	Name string
	Type Type
}

type Type string

const IntType Type = "int"

type Block struct {
	Statements []Statement
}

type Statement interface {
	statementNode()
}

type LetStmt struct {
	Name  string
	Type  Type
	Value Expression
}

func (LetStmt) statementNode() {}

type ReturnStmt struct {
	Value Expression
}

func (ReturnStmt) statementNode() {}

type ExprStmt struct {
	Expr Expression
}

func (ExprStmt) statementNode() {}

type Expression interface {
	expressionNode()
}

type IdentExpr struct {
	Name string
}

func (IdentExpr) expressionNode() {}

type IntLiteral struct {
	Value string
}

func (IntLiteral) expressionNode() {}

type CallExpr struct {
	Callee string
	Args   []Expression
}

func (CallExpr) expressionNode() {}

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (BinaryExpr) expressionNode() {}
