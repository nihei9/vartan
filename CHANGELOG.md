# Changelog

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
