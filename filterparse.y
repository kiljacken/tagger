%{
package tagger
%}

%union{
	filter Filter
	tag string
	val int
	comp Comparator
}

%token TAG VAL COMP
%token AND OR
%token LPAREN RPAREN

%type <filter> expr paren and_expr or_expr comp tag
%type <val> VAL
%type <tag> TAG
%type <comp> COMP

%%

goal:
	expr { yylex.(*lex).filter = $1 }

expr:
	paren
|	and_expr
|	or_expr
|	comp
|	tag

paren:
	LPAREN expr RPAREN { $$ = $2 }

and_expr:
	and_expr AND expr
	{
		$$ = AndFilter{Filters: append($1.(AndFilter).Filters, $3)}	
	}
|	expr AND expr
	{
		$$ = AndFilter{Filters: []Filter{$1, $3}}
	}

or_expr:
	or_expr OR expr
	{
		$$ = OrFilter{Filters: append($1.(OrFilter).Filters, $3)}	
	}
|	expr OR expr
	{
		$$ = OrFilter{Filters: []Filter{$1, $3}}
	}

comp:
	TAG COMP VAL
	{
		$$ = ComparinsonFilter{Name: $1, Value: $3, Function: $2}
	}

tag:
	TAG
	{
		$$ = NameFilter{Name: $1}
	}

%%

