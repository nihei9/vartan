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
$ vartan compile -g expr.vr -o expr.json
16 conflicts
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
# Class

LALR(1)

# Conflicts

16 conflicts were detected.

# Terminals

   1  - - <eof>
   2  - - error
   3  - - x_1 (\()
   4  - - x_2 (\))
   5  - - x_3 (,)
   6  - - ws
   7  - - int
   8  - - id
   9  2 l add (+)
  10  2 l sub (-)
  11  1 l mul (*)
  12  1 l div (/)

# Productions

   1  - - expr' → expr
   2  2 l expr → expr + expr
   3  2 l expr → expr - expr
   4  1 l expr → expr * expr
   5  1 l expr → expr / expr
   6  - - expr → func_call
   7  - - expr → int
   8  - - expr → id
   9  - - expr → \( expr \)
  10  - - func_call → id \( args \)
  11  - - func_call → id \( \)
  12  - - args → args , expr
  13  - - args → expr

# States

## State 0

   1 expr' → ・ expr

shift     3 on \(
shift     4 on int
shift     5 on id
goto      1 on expr
goto      2 on func_call


## State 1

   1 expr' → expr ・
   2 expr → expr ・ + expr
   3 expr → expr ・ - expr
   4 expr → expr ・ * expr
   5 expr → expr ・ / expr

shift     6 on +
shift     7 on -
shift     8 on *
shift     9 on /
reduce    1 on <eof>


## State 2

   6 expr → func_call ・

reduce    6 on <eof>, \), ,, +, -, *, /

...
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
	treeAct := NewSyntaxTreeActionSet(gram, true, false)
	p, err := NewParser(toks, gram, SemanticAction(treeAct))
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
	PrintTree(os.Stdout, treeAct.AST())
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
