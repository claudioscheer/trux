package ast

import (
	"fmt"

	"github.com/claudioscheer/trux/internal/token"
)

type Program struct {
	PackageName string
	Imports     []*ImportDecl
	Functions   []*FuncDecl
}

type ImportDecl struct {
	Pos  token.Position
	Path string
}

type FuncDecl struct {
	Pos        token.Position
	Name       string
	Public     bool
	Params     []Param
	ReturnType Type
	Body       Block
}

type Param struct {
	Name string
	Type Type
}

type Type interface {
	fmt.Stringer
	typeNode()
}

type ScalarType string

const (
	IntType    ScalarType = "int"
	FloatType  ScalarType = "float"
	StringType ScalarType = "string"
	BoolType   ScalarType = "bool"
)

func (t ScalarType) String() string { return string(t) }

func (ScalarType) typeNode() {}

type ArrayType struct {
	Length int
	Elem   Type
}

func (t *ArrayType) String() string {
	return fmt.Sprintf("[%d]%s", t.Length, t.Elem)
}

func (*ArrayType) typeNode() {}

type SliceType struct {
	Elem Type
}

func (t *SliceType) String() string {
	return fmt.Sprintf("[]%s", t.Elem)
}

func (*SliceType) typeNode() {}

type ListType struct {
	Elem Type
}

func (t *ListType) String() string {
	return fmt.Sprintf("list[%s]", t.Elem)
}

func (*ListType) typeNode() {}

func TypeEqual(left Type, right Type) bool {
	switch left := left.(type) {
	case ScalarType:
		right, ok := right.(ScalarType)
		return ok && left == right
	case *ArrayType:
		right, ok := right.(*ArrayType)
		return ok && left.Length == right.Length && TypeEqual(left.Elem, right.Elem)
	case *SliceType:
		right, ok := right.(*SliceType)
		return ok && TypeEqual(left.Elem, right.Elem)
	case *ListType:
		right, ok := right.(*ListType)
		return ok && TypeEqual(left.Elem, right.Elem)
	default:
		return left == nil && right == nil
	}
}

func ElementType(typ Type) (Type, bool) {
	switch typ := typ.(type) {
	case *ArrayType:
		return typ.Elem, true
	case *SliceType:
		return typ.Elem, true
	case *ListType:
		return typ.Elem, true
	default:
		return nil, false
	}
}

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

type IndexAssignStmt struct {
	Start  token.Position
	Target *IndexExpr
	Value  Expression
}

func (s *IndexAssignStmt) Pos() token.Position { return s.Start }

func (*IndexAssignStmt) statementNode() {}

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

type ArrayLiteral struct {
	Start    token.Position
	Type     Type
	Elements []Expression
}

func (e *ArrayLiteral) Pos() token.Position { return e.Start }

func (*ArrayLiteral) expressionNode() {}

type ListLiteral struct {
	Start    token.Position
	Type     Type
	Elements []Expression
}

func (e *ListLiteral) Pos() token.Position { return e.Start }

func (*ListLiteral) expressionNode() {}

type MakeExpr struct {
	Start token.Position
	Type  Type
	Len   Expression
}

func (e *MakeExpr) Pos() token.Position { return e.Start }

func (*MakeExpr) expressionNode() {}

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

type IndexExpr struct {
	Start      token.Position
	Collection Expression
	Index      Expression
}

func (e *IndexExpr) Pos() token.Position { return e.Start }

func (*IndexExpr) expressionNode() {}

type SliceExpr struct {
	Start      token.Position
	Collection Expression
	StartIndex Expression
	EndIndex   Expression
}

func (e *SliceExpr) Pos() token.Position { return e.Start }

func (*SliceExpr) expressionNode() {}
