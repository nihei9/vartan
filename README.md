# vartan

vartan provides a compiler that generates a LALR(1) or SLR(1) parsing table and a driver for golang.

[![Test](https://github.com/nihei9/vartan/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/nihei9/vartan/actions/workflows/test.yml)

## Status

🚧 Now Developing

## Installation

```sh
$ go install github.com/nihei9/vartan/cmd/vartan@latest
```

## Usage

vartan uses BNF-like DSL to define your grammar. As an example, let's write a grammar that represents a simple expression.

```
expr
    : expr add_op term
    | term
    ;
term
    : term mul_op factor
    | factor
    ;
factor
    : number
    | id
    ;

whitespaces: "[\u{0009}\u{0020}]+" #skip;
number: "[0-9]+";
id: "[A-Za-z_]+";
add_op: '+';
mul_op: '*';
```

Save the above grammar to a file in UTF-8. In this explanation, the file name is `expr.vr`.

Next, generate a parsing table using `vartan compile` command.

```sh
$ vartan compile -g expr.vr -o expr.json
```

If you want to make sure that the grammar behaves as expected, you can use `vartan parse` command to try parse without implementing a driver.

⚠️ An encoding that `vartan parse` command and the driver can handle is only UTF-8.

```sh
$ echo -n 'foo + bar * baz * 100' | vartan parse expr.json
expr
├─ expr
│  └─ term
│     └─ factor
│        └─ id "foo"
├─ add_op "+"
└─ term
   ├─ term
   │  ├─ term
   │  │  └─ factor
   │  │     └─ id "bar"
   │  ├─ mul_op "*"
   │  └─ factor
   │     └─ id "baz"
   ├─ mul_op "*"
   └─ factor
      └─ number "100"
```

When `vartan parse` command successfully parses the input data, it prints a CST or an AST (if any).

## Debug

`vartan compile` command also generates a description file having `-description.json` suffix along with a parsing table. This file describes each state in the parsing table in detail. If your grammar contains conflicts, see `Conflicts` and `States` sections of this file. Using `vartan show` command, you can see the description file in a readable format.

```sh
$ vartan show expr-description.json
# Class

LALR(1)

# Conflicts

No conflict was detected.

# Terminals

   1  - - <eof>
   2  - - error
   3  - - whitespaces
   4  - - number
   5  - - id
   6  - - add_op (+)
   7  - - mul_op (*)

# Productions

   1  - - expr' → expr
   2  - - expr → expr + term
   3  - - expr → term
   4  - - term → term * factor
   5  - - term → factor
   6  - - factor → number
   7  - - factor → id

# States

## State 0

   1 expr' → ・ expr

shift     4 on number
shift     5 on id
goto      1 on expr
goto      2 on term
goto      3 on factor

...
```
