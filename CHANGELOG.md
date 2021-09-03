# Changelog

## v0.3.0

* [7271e46b](https://github.com/nihei9/vartan/commit/7271e46bbcb11acf860c91eddfe12dd7eed5ccad) - Add `error` symbol and `#recover` directive to recover from an error state.
* [a769f496](https://github.com/nihei9/vartan/commit/a769f496ecba60a73d74c445f5894ce52be800ee) - Add an `#alias` directive to define a user-friendly name of a terminal.
* [4fda9eb3](https://github.com/nihei9/vartan/commit/4fda9eb3584cfcfd1e35267526442c022693f7ed) - Support the escape sequecens `\'` and `\\` in a string literal.
* [936b600c](https://github.com/nihei9/vartan/commit/936b600ce23cce4350a730817a067a8926384baf) - Use a pattern string defined by a string literal as its alias.
* [b70f4184](https://github.com/nihei9/vartan/commit/b70f41840819a59f82a37c0da7eddae40fc52aa0), [d904e822](https://github.com/nihei9/vartan/commit/d904e8224505fbbc7ae6f4a412a14096dcb2fde8) - Add show command to print a description file.
* [bb85dcc5](https://github.com/nihei9/vartan/commit/bb85dcc57cc3c0fff6cc9dc09540d58fef400d6f) - Add precedences and associativities to the description file.
* [3584af7b](https://github.com/nihei9/vartan/commit/3584af7bc0bdf7388bc43aaa60d432b98afb752d) - Add `#prec` directive to set precedence and associativity of productions.
* [ccf0123d](https://github.com/nihei9/vartan/commit/ccf0123d7f1b88ee7cdd4e2ea15ab9e94457538a) - Remove the expected terminals field from the parsing table. The driver searches the expected terminals corresponding to each state if necessary.
* [8832b64b](https://github.com/nihei9/vartan/commit/8832b64b4227245e45f9a24d543c1b80168c489d), [0bcf9458](https://github.com/nihei9/vartan/commit/0bcf94582b4c33de212b948cf512267f9af8eb74) - Support LAC (lookahead correction).
* [f4e3fef0](https://github.com/nihei9/vartan/commit/f4e3fef07e8e38e37e63989254718e6c4cb543a9) - Make semantic actions user-configurable.
* [ad35bf24](https://github.com/nihei9/vartan/commit/ad35bf24d80c36b3847538cf846d35de7751f7f2) - Use the LALR by default when using _grammar.Compile_ instead of the CLI.

[Changes](https://github.com/nihei9/vartan/compare/v0.2.0...v0.3.0)

## v0.2.0

* [00f8b09](https://github.com/nihei9/vartan/commit/00f8b091a9f1eb3ed0348900784be07c326c0dc1) - Support LALR(1) class.
* [05738fa](https://github.com/nihei9/vartan/commit/05738faa189e50b6c0ecc52f0e2dbad6bcedb218) - Print a stack trace only when a panic occured.
* [118732e](https://github.com/nihei9/vartan/commit/118732eccef2350bf4e20e389b35b2433613b1ab) - Fix panic on a syntax error.
* [6c2036e](https://github.com/nihei9/vartan/commit/6c2036e86fc37a5361d6daf64b914f1af66559ef) - Fix indents of a tree.
* [94e2400](https://github.com/nihei9/vartan/commit/94e2400aa8e6017165ab22ba5f2f70a6d0682f63) - Resolve conflicts by default rules. When a shift/reduce conflict occurred, we prioritize the shift action, and when a reduce/reduce conflict occurred, we prioritize the production defined earlier in the grammar file.
* [4d879b9](https://github.com/nihei9/vartan/commit/4d879b95d5368d578a39baaefba0de743a764105) - Support `%left` and `%right` to specify precedences and associativities.
* [02674d7](https://github.com/nihei9/vartan/commit/02674d7264aea363a8f7b7839ab77ce64ba720db) - Add a column number to an error message.
* [dc78a8b](https://github.com/nihei9/vartan/commit/dc78a8b9b9496a6e26ac8ebb925bd708a83af307) - Add a column number to a token.
* [6cfbd0a](https://github.com/nihei9/vartan/commit/6cfbd0a8bb969d440bbf836947ae4a12cda56ab3) - Fix panic on no productions.

[Changes](https://github.com/nihei9/vartan/compare/v0.1.1...v0.2.0)

## v0.1.1

* [bb878f9](https://github.com/nihei9/vartan/commit/bb878f980b26f4a90a0ba7ec18e6a044a04e7d14) - Fix the name of the EOF symbol in the description file. The EOF is displayed as _\<EOF>_, not _e1_.
* [c14ae41](https://github.com/nihei9/vartan/commit/c14ae41955cdfc141208a4518f257bd3fa138a47) - Generate an AST and a CST only when parser options are enabled.
* [b8ef796](https://github.com/nihei9/vartan/commit/b8ef7961255ed1d9ef5c51a92f1832c99c6d89cd) - Add `--cst` option to `vartan parse` command.
* [2bf3786](https://github.com/nihei9/vartan/commit/2bf3786801cd6727e3f28d0a6aeb7ec375eb1aa7) - Avoid the growth of slices when constructing trees.
* [7b4ed66](https://github.com/nihei9/vartan/commit/7b4ed6608ea338c77e89b06bb20efae15491fcbc) - Add `--only-parse` option to `vartan parse` command.
* [3d417d5](https://github.com/nihei9/vartan/commit/3d417d5181bd373cbc6e9734ee709c588600a457) - Print a stack trace on panic.

[Changes](https://github.com/nihei9/vartan/compare/v0.1.0...v0.1.1)

## v0.1.0

* vartan v0.1.0, this is the first release, supports the following features.
  * `vartan compile`: SLR parsing table generation
    * Definitions of grammars by simple DSL
    * Automatic AST construction by `#ast` directive
  * `vartan parse`: Driver for the parsing table

[Commits](https://github.com/nihei9/vartan/commits/v0.1.0)
