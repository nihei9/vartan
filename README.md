# vartan

vartan is a parser generator for golang and supports LALR(1) and SLR(1). vartan also provides a command to perform syntax analysis to allow easy debugging of your grammar.

[![Test](https://github.com/nihei9/vartan/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/nihei9/vartan/actions/workflows/test.yml)

## Installation

Compiler:

```sh
$ go install github.com/nihei9/vartan/cmd/vartan@latest
```

Code Generator:

```sh
$ go install github.com/nihei9/vartan/cmd/vartan-go@latest
```

## Usage

### 1. Define your grammar

vartan uses BNF-like DSL to define your grammar. As an example, let's write a grammar that represents a simple expression.

```
%name expr

%left mul div
%left add sub

expr
	: expr add expr
	| expr sub expr
	| expr mul expr
	| expr div expr
	| func_call
	| int
	| id
	| '(' expr ')' #ast expr
	;
func_call
	: id '(' args ')' #ast id args
	| id '(' ')'      #ast id
	;
args
	: args ',' expr #ast args... expr
	| expr
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
int
	: "0|[1-9][0-9]*";
id
	: "[A-Za-z_][0-9A-Za-z_]*";
add
	: '+';
sub
	: '-';
mul
	: '*';
div
	: '/';
```

Save the above grammar to a file in UTF-8. In this explanation, the file name is `expr.vr`.

⚠️ The input file must be encoded in UTF-8.

### 2. Compile the grammar

Next, generate a parsing table using `vartan compile` command.

```sh
$ vartan compile expr.vr -o expr.json
```

### 3. Debug

#### 3.1. Parse

If you want to make sure that the grammar behaves as expected, you can use `vartan parse` command to try parse without implementing a driver.

⚠️ An encoding that `vartan parse` command and the driver can handle is only UTF-8.

```sh
$ echo -n 'foo(10, bar(a)) + 99 * x' | vartan parse expr.json
expr
├─ expr
│  └─ func_call
│     ├─ id "foo"
│     └─ args
│        ├─ expr
│        │  └─ int "10"
│        └─ expr
│           └─ func_call
│              ├─ id "bar"
│              └─ args
│                 └─ expr
│                    └─ id "a"
├─ add "+"
└─ expr
   ├─ expr
   │  └─ int "99"
   ├─ mul "*"
   └─ expr
      └─ id "x"
```

When `vartan parse` command successfully parses the input data, it prints a CST or an AST (if any).

#### 3.2. Resolve conflicts

`vartan compile` command also generates a description file having `-description.json` suffix along with a parsing table. This file describes each state in the parsing table in detail. If your grammar contains conflicts, see `Conflicts` and `States` sections of this file. Using `vartan show` command, you can see the description file in a readable format.

```sh
$ vartan show expr-description.json
```

### 4. Generate a parser

Using `vartan-go` command, you can generate a source code of a parser to recognize your grammar.

```sh
$ vartan-go expr.json
```

Then you will get the following files.

* `expr_parser.go`
* `expr_lexer.go`
* `expr_semantic_action.go`

You need to implement a driver to use the parser. An example is below.

```go
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	toks, err := NewTokenStream(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	gram := NewGrammar()
	tb := NewDefaultSyntaxTreeBuilder()
	p, err := NewParser(toks, gram, SemanticAction(NewASTActionSet(gram, tb)))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = p.Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	synErrs := p.SyntaxErrors()
	if len(synErrs) > 0 {
		for _, synErr := range synErrs {
			printSyntaxError(os.Stderr, synErr, gram)
		}
		os.Exit(1)
	}
	fmt.Println("accepted")
	PrintTree(os.Stdout, tb.Tree())
}

func printSyntaxError(w io.Writer, synErr *SyntaxError, gram Grammar) {
	var msg string
	tok := synErr.Token
	switch {
	case tok.EOF():
		msg = "<eof>"
	case tok.Invalid():
		msg = fmt.Sprintf("'%v' (<invalid>)", string(tok.Lexeme()))
	default:
		if alias := gram.TerminalAlias(tok.TerminalID()); alias != "" {
			msg = fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), alias)
		} else {
			msg = fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), gram.Terminal(tok.TerminalID()))
		}
	}
	fmt.Fprintf(w, "%v:%v: %v: %v", synErr.Row+1, synErr.Col+1, synErr.Message, msg)

	if len(synErr.ExpectedTerminals) > 0 {
		fmt.Fprintf(w, "; expected: %v", synErr.ExpectedTerminals[0])
		for _, t := range synErr.ExpectedTerminals[1:] {
			fmt.Fprintf(w, ", %v", t)
		}
	}

	fmt.Fprintf(w, "\n")
}
```

Please save the above source code to `main.go` and create a directory structure like the following.

```
/project_root
├── expr_parser.go
├── expr_lexer.go
├── expr_semantic_action.go
└── main.go (the driver you implemented)
```

Now, you can perform the syntax analysis.

```sh
$ echo -n 'foo+99' | go run .
accepted
expr
├─ expr
│  └─ id "foo"
├─ add "+"
└─ expr
   └─ int "99"
```

```sh
$ echo -n 'foo+99?' | go run .
1:7: unexpected token: '?' (<invalid>); expected: <eof>, +, -, *, /
exit status 1
```

## Vartan syntax

### Grammar name

A grammar name `%name <Identifier>` is an identifier that represents a grammar name. For now, this identifier is used as a file name generated like _<grammar-name>\_parser.go_.

### Production rules

A production rule consists of a non-terminal symbol and sequences of symbols the non-terminal symbol derives. The first production rule will be the start production rule.

Production rule:

```
<non-terminal-symbol>
	: <alternative-1>
	| <alternative-2>
	| ...
	| <alternative-N>
	;
```

or

```
<terminal-symbol>
	: <pattern-or-string-literal>
	;
```

Alternative:

```
<element-1> <element-2> ... <element-N>
```

An element an alternative contains is a terminal symbol, a non-terminal symbol, a pattern, or a string literal.

You can define terminal symbols in the same grammar as non-terminal symbols.

If a production rule satisfies all of the following conditions, it is considered to define a terminal symbol.

* A production rule has only one alternative.
* the alternative has only one pattern or string literal.

Fragment:

```
fragment <terminal-symbol>
	: <pattern-or-string>
	;
```

Using the `fragment` keyword, you can also define a fragment that represents a part of a pattern. You can use a fragment by embedding it into a pattern like `"\f{some_fragment}"`.

### Types

#### Identifier

An identifier is a string that satisfies all of the following rules:

* Contains only lowercase letters (`a`-`z`), numbers (`0`-`9`), and underscores (`_`).
* The first letter is only a lowercase letter.
* The last letter is only a lowercase letter or a number.

examples:

`expression`, `if_statement`, `parameter1`

#### Pattern

A pattern is a string enclosed with `"` and represents a regular expression. A pattern that appears in production rules is used in lexical analysis. For more information on the syntax of regular expressions, please see [maleeni's documents](https://github.com/nihei9/maleeni/blob/main/README.md). vartan uses [maleeni](https://github.com/nihei9/maleeni) as a lexer.

examples:

`"if"`, `"\+"`, `"[A-Za-z_][0-9A-Za-z_]*"`, `[\u{0009}\u{0020}]+`

#### String literal

A string literal is a string enclosed with `'`. A string literal is interpreted literally, not as a regular expression.

examples:

`'if'`, `'+'`, `'=='`

### Directives for non-terminal symbols

#### `#ast {<symbol-or-label: Identifier>}`

A `#ast` directive allows you to define a structure of an AST (Abstract Syntax Tree).

example 1:

Consider a grammar that accepts comma-separated list of integers. You can avoid including brackets and commas in an AST by specifying only the necessary symbols int the `#ast` directive parameters. Also, you can flatten an AST using `...` operator. `...` operator expands child nodes of a specified symbol.

```
%name example

list
	: '[' elems ']' #ast elems...
	;
elems
	: elems ',' int #ast elems... int
	| int
	;

ws #skip
   : "[\u{0009}\u{0020}]+";
int
	: "0|[1-9][0-9]*";
```

The above grammar generates an AST as follows:

```
$ echo -n '[1, 2, 3]' | vartan parse example.json
list
├─ int "1"
├─ int "2"
└─ int "3"
```

example 2:

Consider a grammar that accepts ternary-if expression (`<condition> ? <value if true> : <value if false>`). As this grammar contains two `int` symbols, you need to add labels to each symbol to distinguish them. A label consists of `@` + an identifier.

```
%name example

if_expr
	: id '?' int@true ':' int@false #ast id true false
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
id
	: "[a-z_][0-9a-z_]*";
int
	: "0|[1-9][0-9]*";
```

The above grammar generates an AST as follows:

```
$ echo -n 'x? 0 : 99' | vartan parse example.json
if_expr
├─ id "x"
├─ int "0"
└─ int "99"
```

Labels are intended to identify elements in directives. An AST doesn't contain labels.

#### `#prec <symbol: Identifier>`

A `#prec` directive gives alternatives the same precedence as `symbol`.

See [Operator precedence and associativity](#operator-precedence-and-associativity) section for more details on the `#prec` directive.

#### `#recover`

A parser transitions to an error state when an unexpected token appears. By default, the parser recovers from the error state when it shifts three tokens after going to the error state.

When the parser reduces a non-terminal symbol having a `#recover` directive, the parser recovers from the error state.

See [Error recovery](#error-recovery) section for more details on the `#recover` directive.

### Directives for terminal symbols

#### `#alias <alias: String>`

An `#alias` directive aliases for a terminal symbol. You can use the alias in error messages, for example.

example:

```
%name example

s
	: id
	;

id #alias 'identifier'
	: "[A-Za-z_][0-9A-Za-z_]*";
```

#### `#mode {<mode-name: Identifier>}`, `#push <mode-name: Identifier>`, and `#pop`

A terminal symbol with a `#mode` directive is recognized only on a specified mode (`mode-name`). The mode is lexer state, which a `#push` directive and a `#pop` directive can switch.

When the parser shifts a terminal symbol having the `#push` directive, the current mode of the lexer will change to the specified mode (`mode-name`). Using the `#pop` directive, you can make the lexer revert to the previous mode.

example:

```
%name example

tag_pairs
	: tag_pairs tag_pair
	| tag_pair
	;
tag_pair
	: open_tag tag_pairs close_tag
	| open_tag text close_tag
	;
open_tag
	: tag_open name tag_close
	;
close_tag
	: tag_open closing_mark name tag_close
	;

ws #skip
	: "[\u{0009}\u{000A}\u{000D}\u{0020}]+";
text
	: "([^<]|\\<)+";
tag_open #push tag
	: '<';
tag_close #mode tag #pop
	: '>';
closing_mark #mode tag
	: '/';
name #mode tag
	: "[a-z_-][0-9a-z_-]*";
```

The above grammar accepts XML-like texts such as the following:

```
<assistant-director>
	<name>Walter Skinner</name>
	<born>June 3, 1952</born>
</assistant-director>
```

#### `#skip`

The parser doesn't shift a terminal symbol having a `#skip` directive. In other words, these terminal symbols are recognized in lexical analysis but not used in syntax analysis. The `#skip` directive helps define delimiters like white spaces.

example:

```
%name example

s
	: foo bar
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
foo
	: 'foo';
bar
	: 'bar';
```

The above grammar accepts the following input:

```
foo    bar
```

```
foobar
```

### Operator precedence and associativity

`%left` and `%right` allow you to define precedence and associativiry of symbols. `%left`/`%right` each assign the left/right associativity to symbols.

When the right-most terminal symbol of an alternative has precedence or associativity defined explicitly, the alternative inherits its precedence and associativity.

`#prec` directive assigns the same precedence as a specified symbol to an alternative.

The grammar for simple four arithmetic operations and assignment expression can be defined as follows:

```
%name example

%left mul div
%left add sub
%right assign

expr
	: expr add expr
	| expr sub expr
	| expr mul expr
	| expr div expr
	| id assign expr
	| int
	| sub int #prec mul // This `sub` means a unary minus symbol.
	| id
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
int
	: "0|[1-9][0-9]*";
id
	: "[a-z_][0-9a-z_]*";
add
	: '+';
sub
	: '-';
mul
	: '*';
div
	: '/';
assign
	: '=';
```

`%left` and `%right` can appear multiple times, and the first symbols applied to will have the highest precedence. That is, `mul` and `div` have the highest precedence, and `assign` has the lowest precedence.

⚠️ In many Yacc-like tools, the last symbols defined have the highest precedence. Not that in vartan, it is the opposite.

When you compile the above grammar, some conflicts occur. However, vartan can resolve the conflicts following `%left`, `%right`, and `#prec`.

```
$ echo -n 'foo = bar = x * -1 / -2' | vartan parse example.json
expr
├─ id "foo"
├─ assign "="
└─ expr
   ├─ id "bar"
   ├─ assign "="
   └─ expr
      ├─ expr
      │  ├─ expr
      │  │  └─ id "x"
      │  ├─ mul "*"
      │  └─ expr
      │     ├─ sub "-"
      │     └─ int "1"
      ├─ div "/"
      └─ expr
         ├─ sub "-"
         └─ int "2"
```

Incidentally, using no directives, you can define the above example as the following grammar:

```
%name example

expr
	: id assign expr
	| arithmetic
	;

arithmetic
	: arithmetic add term
	| arithmetic sub term
	| term
	;
term
	: term mul factor
	| term div factor
	| factor
	;
factor
	: int
	| sub int
	| id
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
int
	: "0|[1-9][0-9]*";
id
	: "[a-z_][0-9a-z_]*";
add
	: '+';
sub
	: '-';
mul
	: '*';
div
	: '/';
assign
	: '=';
```

This grammar expresses precedence and associativity by nesting production rules instead of directives. Also, no conflicts occur in compiling this grammar.

However, the more production rules you define, the more time syntax analysis needs. A structure of a mechanically generated parse tree is also more complex, but you can improve the structure of the parse tree using `#ast` directive.

```
$ echo -n 'foo = bar = x * -1 / -2' | vartan parse example.json
expr
├─ id "foo"
├─ assign "="
└─ expr
   ├─ id "bar"
   ├─ assign "="
   └─ expr
      └─ arithmetic
         └─ term
            ├─ term
            │  ├─ term
            │  │  └─ factor
            │  │     └─ id "x"
            │  ├─ mul "*"
            │  └─ factor
            │     ├─ sub "-"
            │     └─ int "1"
            ├─ div "/"
            └─ factor
               ├─ sub "-"
               └─ int "2"
```

### Error recovery

By default, a parser will stop syntax analysis on a syntax error. If you want to continue semantic actions after syntax errors occur, you can use an `error` symbol and a `#recover` directive.

When a syntax error occurs, the parser pops states from a state stack. If a state containing the `error` symbol appears, a parser stops popping the states. It then shifts the `error` symbol and resumes syntax analysis.

Consider grammar of simple assignment statements.

```
%name example

statements
	: statements statement #ast statements... statement
	| statement            #ast statement
	;
statement
	: name '=' int ';' #ast name int
	| error ';'        #recover
	;

ws #skip
	: "[\u{0009}\u{0020}]";
name
	: "[a-z_][0-9a-z_]*";
int
	: "0|[1-9][0-9]*";
```

The alternative `error ';'` traps a syntax error and discards symbols until the parser shifts `';'`. When the parser shifts `';'` and reduces `statement`, the parser will recover from an error state immediately by the `#recover` directive.

In the following example, you can see the parser print syntax error messages and an AST. This result means the parser had continued syntax analysis and semantic actions even if syntax errors occurred.

```
$ echo -n 'x; x =; x = 1;' | vartan parse example.json
1:2: unexpected token: ';' (x_2); expected: =
1:7: unexpected token: ';' (x_2); expected: int

statements
├─ statement
│  ├─ !error
│  └─ x_2 ";"
├─ statement
│  ├─ !error
│  └─ x_2 ";"
└─ statement
   ├─ name "x"
   └─ int "1"
```
