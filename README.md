# vartan

vartan is an LALR(1) parser generator for golang. vartan also provides a command to perform syntax analysis to allow easy debugging of your grammar.

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
#name expr;

#prec (
	#left mul div
	#left add sub
);

expr
	: expr add expr
	| expr sub expr
	| expr mul expr
	| expr div expr
	| func_call
	| int
	| id
	| l_paren expr r_paren #ast expr
	;
func_call
	: id l_paren args r_paren #ast id args
	| id l_paren r_paren      #ast id
	;
args
	: args comma expr #ast args... expr
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
l_paren
	: '(';
r_paren
	: ')';
comma
	: ',';
```

Save the above grammar to a file in UTF-8. In this explanation, the file name is `expr.vartan`.

⚠️ The input file must be encoded in UTF-8.

### 2. Compile the grammar

Next, generate a parsing table using `vartan compile` command.

```sh
$ vartan compile expr.vartan -o expr.json
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

`vartan compile` command also generates a report named `*-report.json`. This file describes each state in the parsing table in detail. If your grammar contains conflicts, see `Conflicts` and `States` sections of this file. Using `vartan show` command, you can see the report in a readable format.

```sh
$ vartan show expr-report.json
```

### 4. Test

`vartan test` command allows you to test whether your grammar recognizes an input text as a syntax tree with an expected structure. To do so, you need to define a test case as follows.

```
This is an example.
---
a / b * 100
---
(expr
	(expr
		(expr (id 'a'))
		(div '/')
		(expr (id 'b')))
	(mul '*')
	(expr (int '100')))
```

The test case consists of a description, an input text, and a syntax tree you expect. Each part is separated by the delimiter `---`. The syntax tree is represented by the syntax like an [S-expression](https://en.wikipedia.org/wiki/S-expression).

A text of a token is represented by a string enclosed in `'` or `"`. Within `"..."`, characters can alose be represented by Unicode code points (for instance, `\u{000A}` is LF).

Save the above test case to `test.txt` file and run the following command.

```sh
$ vartan test expr.vartan test.txt
Passed test.txt
```

When you specify a directory as the 2nd argument of `vartan test` command, it will run all test cases in the directory.

### 5. Generate a parser

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
		if term := gram.Terminal(tok.TerminalID()); term != "" {
			msg = fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), term)
		} else {
			msg = fmt.Sprintf("'%v'", string(tok.Lexeme()))
		}
	}
	fmt.Fprintf(w, "%v:%v: %v: %v", synErr.Row+1, synErr.Col+1, synErr.Message, msg)

	if len(synErr.ExpectedTerminals) > 0 {
		fmt.Fprintf(w, ": expected: %v", synErr.ExpectedTerminals[0])
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
1:7: unexpected token: '?' (<invalid>): expected: <eof>, add, sub, mul, div
exit status 1
```

## Vartan syntax

### Grammar name

A grammar name `#name <Identifier>` is an identifier that represents a grammar name. For now, this identifier is used as a file name generated like _<grammar-name>\_parser.go_.

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

An element an alternative contains is a terminal symbol or a non-terminal symbol.

If a production rule satisfies all of the following conditions, it is considered to define a terminal symbol.

* A rule has only one alternative.
* The alternative has only one pattern or string literal.

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

* Contains only the lower-case letters (`a`-`z`), the digits (`0`-`9`), and the underscore (`_`).
* The first letter is only the lower-case letters.
* The last letter is only the lower-case letters or the digits.

examples:

`expression`, `if_statement`, `parameter1`

#### Pattern

A pattern is a string enclosed with `"` and represents a regular expression. A pattern that appears in production rules is used in lexical analysis. For more information on the syntax of regular expressions, please see [Regular Expression](#regular-expression).

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
#name example;

list
	: l_bracket elems r_bracket #ast elems...
	;
elems
	: elems comma int #ast elems... int
	| int
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
l_bracket
	: '[';
r_bracket
	: ']';
comma
	: ',';
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
#name example;

if_expr
	: id q int@true colon int@false #ast id true false
	;

ws #skip
	: "[\u{0009}\u{0020}]+";
q
	: '?';
colon
	: ':';
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

#### `#mode {<mode-name: Identifier>}`, `#push <mode-name: Identifier>`, and `#pop`

A terminal symbol with a `#mode` directive is recognized only on a specified mode (`mode-name`). The mode is lexer state, which a `#push` directive and a `#pop` directive can switch.

When the parser shifts a terminal symbol having the `#push` directive, the current mode of the lexer will change to the specified mode (`mode-name`). Using the `#pop` directive, you can make the lexer revert to the previous mode.

example:

```
#name example;

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
#name example;

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

`#left` and `#right` directives allow you to define precedence and associativiry of symbols. `#left`/`#right` each assign the left/right associativity to symbols.

If you want to change precedence, `#assign` directive helps you. `#assign` directive changes only precedence, not associativity.

When the right-most terminal symbol of an alternative has precedence or associativity defined explicitly, the alternative inherits its precedence and associativity.

`#prec` directive assigns the same precedence as a specified symbol to an alternative and disables associativity.

You can define an ordered symbol with the form `$<ID>`. The ordered symbol is an identifier having only precedence, and you can use it in `#prec` directive applied to an alternative. The ordered symbol helps you to resolve shift/reduce conflicts without terminal symbol definitions.

The grammar for simple four arithmetic operations and assignment expression can be defined as follows:

```
#name example;

#prec (
	#assign $uminus
	#left mul div
	#left add sub
	#right assign
);

expr
	: expr add expr
	| expr sub expr
	| expr mul expr
	| expr div expr
	| id assign expr
	| int
	| sub int #prec $uminus // This `sub` means a unary minus symbol.
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

`#left` and `#right` can appear multiple times, and the first symbols applied to will have the highest precedence. That is, `mul` and `div` have the highest precedence, and `assign` has the lowest precedence.

⚠️ In many Yacc-like tools, the last symbols defined have the highest precedence. Not that in vartan, it is the opposite.

When you compile the above grammar, some conflicts occur. However, vartan can resolve the conflicts following `#left`, `#right`, and `#prec`.

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
#name example;

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

Consider grammar of simple equivalent expressions.

```
#name example;

eq_exprs
	: eq_exprs eq_expr #ast eq_exprs... eq_expr
	| eq_expr
	;
eq_expr
	: name eq int semi_colon #ast name int
	| error semi_colon       #recover
	;

ws #skip
	: "[\u{0009}\u{0020}]";
eq
	: '=';
semi_colon
	: ';';
name
	: "[a-z_][0-9a-z_]*";
int
	: "0|[1-9][0-9]*";
```

The alternative `error semi_colon` traps a syntax error and discards symbols until the parser shifts `';'`. When the parser shifts `';'` and reduces `eq_expr`, the parser will recover from an error state immediately by the `#recover` directive.

In the following example, you can see the parser print syntax error messages and an AST. This result means the parser had continued syntax analysis and semantic actions even if syntax errors occurred.

```
$ vartan compile example.vartan -o example.json
$ echo -n 'x; x =; x = 1;' | vartan parse example.json
eq_exprs
├─ eq_expr
│  ├─ error
│  └─ semi_colon ";"
├─ eq_expr
│  ├─ error
│  └─ semi_colon ";"
└─ eq_expr
   ├─ name "x"
   └─ int "1"
1:2: unexpected token: ';' (semi_colon): expected: eq
1:7: unexpected token: ';' (semi_colon): expected: int
```

### Regular Expression

⚠️ vartan doesn't allow you to use some code points. See [Unavailable Code Points](#unavailable-code-points).

#### Composites

Concatenation and alternation allow you to combine multiple characters or multiple patterns into one pattern.

| Pattern    | Matches        |
|------------|----------------|
| `abc`      | `abc`          |
| `abc\|def` | `abc` or `def` |

#### Single Characters

In addition to using ordinary characters, there are other ways to represent a single character:

* dot expression
* bracket expressions
* code point expressions
* character property expressions
* escape sequences

##### Dot Expression

The dot expression matches any one chracter.

| Pattern | Matches           |
|---------|-------------------|
| `.`     | any one character |

##### Bracket Expressions

The bracket expressions are represented by enclosing characters in `[ ]` or `[^ ]`. `[^ ]` is negation of `[ ]`. For instance, `[ab]` matches one of `a` or `b`, and `[^ab]` matches any one character except `a` and `b`.

| Pattern  | Matches                                          |
|----------|--------------------------------------------------|
| `[abc]`  | `a`, `b`, or `c`                                 |
| `[^abc]` | any one character except `a`, `b`, and `c`       |
| `[a-z]`  | one in the range of `a` to `z`                   |
| `[a-]`   | `a` or `-`                                       |
| `[-z]`   | `-` or `z`                                       |
| `[-]`    | `-`                                              |
| `[^a-z]` | any one character except the range of `a` to `z` |
| `[a^]`   | `a` or `^`                                       |

##### Code Point Expressions

The code point expressions match a character that has a specified code point. The code points consists of a four or six digits hex string.

| Pattern      | Matches                     |
|--------------|-----------------------------|
| `\u{000A}`   | U+000A (LF)                 |
| `\u{3042}`   | U+3042 (hiragana `あ`)      |
| `\u{01F63A}` | U+1F63A (grinning cat `😺`) |

##### Character Property Expressions

The character property expressions match a character that has a specified character property of the Unicode. Currently, vartan supports `General_Category`, `Script`, `Alphabetic`, `Lowercase`, `Uppercase`, and `White_Space`. When you omitted the equal symbol and a right-side value, vartan interprets a symbol in `\p{...}` as the `General_Category` value.

| Pattern                       | Matches                                                |
|-------------------------------|--------------------------------------------------------|
| `\p{General_Category=Letter}` | any one character whose `General_Category` is `Letter` |
| `\p{gc=Letter}`               | the same as `\p{General_Category=Letter}`              |
| `\p{Letter}`                  | the same as `\p{General_Category=Letter}`              |
| `\p{l}`                       | the same as `\p{General_Category=Letter}`              |
| `\p{Script=Latin}`            | any one character whose `Script` is `Latin`            |
| `\p{Alphabetic=yes}`          | any one character whose `Alphabetic` is `yes`          |
| `\p{Lowercase=yes}`           | any one character whose `Lowercase` is `yes`           |
| `\p{Uppercase=yes}`           | any one character whose `Uppercase` is `yes`           |
| `\p{White_Space=yes}`         | any one character whose `White_Space` is `yes`         |

##### Escape Sequences

As you escape the special character with `\`, you can write a rule that matches the special character itself.
The following escape sequences are available outside of bracket expressions.

| Pattern | Matches |
|---------|---------|
| `\.`    | `.`     |
| `\?`    | `?`     |
| `\*`    | `*`     |
| `\+`    | `+`     |
| `\(`    | `(`     |
| `\)`    | `)`     |
| `\[`    | `[`     |
| `\\|`   | `\|`    |
| `\\`    | `\`     |

The following escape sequences are available inside bracket expressions.

| Pattern | Matches |
|---------|---------|
| `\^`    | `^`     |
| `\-`    | `-`     |
| `\]`    | `]`     |

#### Repetitions

The repetitions match a string that repeats the previous single character or group.

| Pattern | Matches          |
|---------|------------------|
| `a*`    | zero or more `a` |
| `a+`    | one or more `a`  |
| `a?`    | zero or one `a`  |

#### Grouping

`(` and `)` groups any patterns.

| Pattern     | Matches                                         |
|-------------|-------------------------------------------------|
| `a(bc)*d`   | `ad`, `abcd`, `abcbcd`, and so on               |
| `(ab\|cd)+` | `ab`, `cd`, `abcd`, `cdab`, `abcdab`, and so on |

#### Unavailable Code Points

Lexical specifications and source files to be analyzed cannot contain the following code points.

When you write a pattern that implicitly contains the unavailable code points, vartan will automatically generate a pattern that doesn't contain the unavailable code points and replaces the original pattern. However, when you explicitly use the unavailable code points (like `\u{U+D800}` or `\p{General_Category=Cs}`), vartan will occur an error.

* surrogate code points: U+D800..U+DFFF
