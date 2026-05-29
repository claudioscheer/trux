package token

type Type string

const (
	Illegal Type = "ILLEGAL"
	EOF     Type = "EOF"

	Ident  Type = "IDENT"
	Int    Type = "INT"
	Float  Type = "FLOAT"
	String Type = "STRING"

	Package    Type = "PACKAGE"
	Func       Type = "FUNC"
	Return     Type = "RETURN"
	Let        Type = "LET"
	If         Type = "IF"
	Else       Type = "ELSE"
	While      Type = "WHILE"
	IntType    Type = "INT_TYPE"
	FloatType  Type = "FLOAT_TYPE"
	StringType Type = "STRING_TYPE"
	BoolType   Type = "BOOL_TYPE"
	True       Type = "TRUE"
	False      Type = "FALSE"
	In         Type = "IN"
	List       Type = "LIST"

	Assign       Type = "="
	Equal        Type = "=="
	NotEqual     Type = "!="
	Less         Type = "<"
	LessEqual    Type = "<="
	Greater      Type = ">"
	GreaterEqual Type = ">="
	Plus         Type = "+"
	Minus        Type = "-"
	Asterisk     Type = "*"
	Slash        Type = "/"

	Comma Type = ","
	Colon Type = ":"

	LParen   Type = "("
	RParen   Type = ")"
	LBrace   Type = "{"
	RBrace   Type = "}"
	LBracket Type = "["
	RBracket Type = "]"
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
	"if":      If,
	"else":    Else,
	"while":   While,
	"int":     IntType,
	"float":   FloatType,
	"string":  StringType,
	"bool":    BoolType,
	"true":    True,
	"false":   False,
	"in":      In,
	"list":    List,
}

func LookupIdent(ident string) Type {
	if typ, ok := keywords[ident]; ok {
		return typ
	}

	return Ident
}
