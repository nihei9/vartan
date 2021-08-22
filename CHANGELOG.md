# Changelog

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
