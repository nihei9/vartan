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

`vartan compile` command also generates a description file having `.desc` suffix along with a parsing table. This file describes each state in the parsing table in detail. If your grammar contains conflicts, see `Conflicts` section of this file.
