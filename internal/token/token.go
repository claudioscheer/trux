package token

type Type string

const (
	Illegal Type = "ILLEGAL"
	EOF     Type = "EOF"

	Ident Type = "IDENT"
	Int   Type = "INT"

	Package Type = "PACKAGE"
	Func    Type = "FUNC"
	Return  Type = "RETURN"
	Let     Type = "LET"
	IntType Type = "INT_TYPE"

	Assign   Type = "="
	Plus     Type = "+"
	Minus    Type = "-"
	Asterisk Type = "*"
	Slash    Type = "/"

	Comma Type = ","

	LParen Type = "("
	RParen Type = ")"
	LBrace Type = "{"
	RBrace Type = "}"
)

type Position struct {
	Offset int
	Line   int
	Column int
}

type Token struct {
	Type   Type
	Lexeme string
	Pos    Position
}

var keywords = map[string]Type{
	"package": Package,
	"func":    Func,
	"return":  Return,
	"let":     Let,
	"int":     IntType,
}

func LookupIdent(ident string) Type {
	if typ, ok := keywords[ident]; ok {
		return typ
	}

	return Ident
}
