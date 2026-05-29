package ast

import "github.com/claudioscheer/trux/internal/token"

type Program struct {
	PackageName string
	Functions   []*FuncDecl
}

type FuncDecl struct {
	Pos        token.Position
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

const (
	IntType    Type = "int"
	FloatType  Type = "float"
	StringType Type = "string"
	BoolType   Type = "bool"
)

type Block struct {
	Statements []Statement
}

type Statement interface {
	Pos() token.Position
	statementNode()
}

type LetStmt struct {
	Start token.Position
	Name  string
	Type  Type
	Value Expression
}

func (s *LetStmt) Pos() token.Position { return s.Start }

func (*LetStmt) statementNode() {}

type ReturnStmt struct {
	Start token.Position
	Value Expression
}

func (s *ReturnStmt) Pos() token.Position { return s.Start }

func (*ReturnStmt) statementNode() {}

type AssignStmt struct {
	Start token.Position
	Name  string
	Value Expression
}

func (s *AssignStmt) Pos() token.Position { return s.Start }

func (*AssignStmt) statementNode() {}

type IfStmt struct {
	Start     token.Position
	Condition Expression
	Then      Block
	Else      *Block
}

func (s *IfStmt) Pos() token.Position { return s.Start }

func (*IfStmt) statementNode() {}

type WhileStmt struct {
	Start     token.Position
	Condition Expression
	Body      Block
}

func (s *WhileStmt) Pos() token.Position { return s.Start }

func (*WhileStmt) statementNode() {}

type ExprStmt struct {
	Expr Expression
}

func (s *ExprStmt) Pos() token.Position { return s.Expr.Pos() }

func (*ExprStmt) statementNode() {}

type Expression interface {
	Pos() token.Position
	expressionNode()
}

type IdentExpr struct {
	Start token.Position
	Name  string
}

func (e *IdentExpr) Pos() token.Position { return e.Start }

func (*IdentExpr) expressionNode() {}

type IntLiteral struct {
	Start token.Position
	Value string
}

func (e *IntLiteral) Pos() token.Position { return e.Start }

func (*IntLiteral) expressionNode() {}

type FloatLiteral struct {
	Start token.Position
	Value string
}

func (e *FloatLiteral) Pos() token.Position { return e.Start }

func (*FloatLiteral) expressionNode() {}

type StringLiteral struct {
	Start token.Position
	Value string
}

func (e *StringLiteral) Pos() token.Position { return e.Start }

func (*StringLiteral) expressionNode() {}

type BoolLiteral struct {
	Start token.Position
	Value bool
}

func (e *BoolLiteral) Pos() token.Position { return e.Start }

func (*BoolLiteral) expressionNode() {}

type CallExpr struct {
	Start  token.Position
	Callee string
	Args   []Expression
}

func (e *CallExpr) Pos() token.Position { return e.Start }

func (*CallExpr) expressionNode() {}

type BinaryExpr struct {
	Start    token.Position
	Left     Expression
	Operator string
	Right    Expression
}

func (e *BinaryExpr) Pos() token.Position { return e.Start }

func (*BinaryExpr) expressionNode() {}
