# Changelog

## v0.6.0

* [acc2df9](https://github.com/nihei9/vartan/commit/acc2df9107e6531529460c7a77da07166d2d45bb) - Stop handling panic to print a stack trace.
* [15ea142](https://github.com/nihei9/vartan/commit/15ea142b25927d6f103ee6ddde4fe8a5e2324831) - Prohibit using escape sequences in string literals.
* [a443f0a](https://github.com/nihei9/vartan/commit/a443f0a1a4b79faa0b0271b72ea4a8e5f3d03ffd) - Prohibit applying the expansion operator to anything other than identifiers.
* [24fd805](https://github.com/nihei9/vartan/commit/24fd80555cb35d9fc63cca9e8697bf156f41780b) - Prohibit applying `#left`, `#right`, `#assign`, and #prec to an error symbol. The shift of the error symbol is an operation forced by the driver. Therefore it is impossible to change this behavior by giving precedence to the error symbol. If we desire to change the precedence of a production rule with the error symbol, we can use #prec directive.
* [b5ad1d3](https://github.com/nihei9/vartan/commit/b5ad1d30df993d68cc64c140bf1005b5490f2605) - Stop supporting SLR(1) and always use LALR(1).
* [def1459](https://github.com/nihei9/vartan/commit/def1459d58bd7c362c626ca2307d83bf576e9dce) - Allows a directory to be specified as the `--output` option for the `vartan compile` command.
* [4c0f7eb](https://github.com/nihei9/vartan/commit/4c0f7eb15db566a19eb753e6ffc3a2354078f5fc) - Rename spec package to `spec/grammar` package.
* [ceb6649](https://github.com/nihei9/vartan/commit/ceb6649d3fb8b85ac8629a65dcfb9533763f4af0) - Add `vartan test` command.
* [1ebed92](https://github.com/nihei9/vartan/commit/1ebed922b8c642b6ff4cec71820f24ceeb331c18) - Support testable tree output in `vartan parse` command.
* [7403c18](https://github.com/nihei9/vartan/commit/7403c18fbc04f3045df1e69b63d7ffd5f04d77db) - Remove the kind field from a node corresponding to an anonymous terminal symbol.
* [52ad315](https://github.com/nihei9/vartan/commit/52ad315a0df8a346953e570e8be472709b81cc6a) - Support the underscore symbol matching any symbols in `vartan test` command.
* [2dd098d](https://github.com/nihei9/vartan/commit/2dd098d1e16bd0b8786ca97ccc7d3b06fa6bc3d1) - Prohibit using a pattern in an alternative. When a syntax error occurs, the parser must provide a user with the names of expected tokens. However, if a pattern appears directly in an alternative, Vartan cannot assign an appropriate name to the pattern. Therefore, this commit prohibits alternatives from containing patterns.
* [4b2cf7b](https://github.com/nihei9/vartan/commit/4b2cf7bc03efe6c4fc0b19ca5da071583bdcb900) - Support testing token's texts in `vartan test` command.
* [017b1b2](https://github.com/nihei9/vartan/commit/017b1b28b3ed67434d1cd58b4cda47bf0a2ed431) - Fix parse error messages for tree parser.
* [2fcab70](https://github.com/nihei9/vartan/commit/2fcab705ddf787fa0f4edab2f2dc7e579ca982d9) - Prohibit error node having children.
* [9033157](https://github.com/nihei9/vartan/commit/90331573a642f1e7e6f3758f1eab8628793bb8f4) - Remove underscore syntax matching any symbol.
  Underscore syntax:
  For instance, a tree `(expr (id 'a') (add '+') (_))` matches both source codes `a + b * c` and `a - b / c`.
  This feature is helpful because it allows you to emphasize the main points of the test by ignoring nodes of no interest. However, we will remove the feature for the time being to reconsider the grammar.
* [d45b860](https://github.com/nihei9/vartan/commit/d45b86063a0e5cdf8cf4efb163637b6d0ee9c5af) - Add tests.
* [5da3885](https://github.com/nihei9/vartan/commit/5da388521510b839406b6232748bc669cda97dec) - Upgrade Go compiler to v1.19.
* [ffa153b](https://github.com/nihei9/vartan/commit/ffa153b220bdbb0c502383026f1e694e088cc2bb) - Update README.
* [b24f61a](https://github.com/nihei9/vartan/commit/b24f61a465d21af404ed647a977160042017e601) - Remove alias system to tidy up the specification.
* [a6001b3](https://github.com/nihei9/vartan/commit/a6001b32cf805c4e72e05adc37ee60272a600bf1) - Remove anonymous symbol system to tidy up the specification.
* [a0aa647](https://github.com/nihei9/vartan/commit/a0aa647770ac6458e9b4749c3efa38d56e2fe4fe) - Move the skip table from lexer-related data to parser-related data.
* [a84350c](https://github.com/nihei9/vartan/commit/a84350cc4a213b74fdd592e8b9eeeb6079d0f5ff) - Split SymbolTable's APIs into reader/writer.
* [f89d021](https://github.com/nihei9/vartan/commit/f89d021bbe134e3efa0d015a41e9712960cdd009) - Import source code of lexer generator from [maleeni](https://github.com/nihei9/maleeni).

[Changes](https://github.com/nihei9/vartan/compare/v0.5.0...v0.6.0)

## v0.5.0

* [f7484ef1](https://github.com/nihei9/vartan/commit/f7484ef11af39585989dbbcad701551c561fa67c) - Add `--json` option to `vartan parse` command to print a syntax tree in JSON format.
* [0eb44f04](https://github.com/nihei9/vartan/commit/0eb44f044b6a4f051126e2e46fd8840dcb105ae9) - Make `#prec` directive change only precedence and not associativity.
* [3eb0e88f](https://github.com/nihei9/vartan/commit/3eb0e88f911386a4e6eca991c1471070596c5554) - Change syntax for top-level directives. `%name`, `%left`, and `%right` changed to `#name`, `#left`, and `#right`, respectively.
* [2438fa44](https://github.com/nihei9/vartan/commit/2438fa4435d6393168412574a3ef94396a4debe5) - Add `#assign` directive to change only precedence.
* [dd5fd337](https://github.com/nihei9/vartan/commit/dd5fd3372cdb53e7a3a36b5ef61b0b0c35023798) - Add the ordered symbol notation `$<Identifier>`.
* [0ff00d27](https://github.com/nihei9/vartan/commit/0ff00d27b2fd524c76fcfac1836b7aad8fe03069) - Change the suffix of a description file from `-description.json` to `-report.json`.
* [0b858d66](https://github.com/nihei9/vartan/commit/0b858d664433d0b11e1d46dbd774e19da5b5a750) - Change the default suffix of a grammar file from `.vr` to `.vartan`.
* [054307b6](https://github.com/nihei9/vartan/commit/054307b6c99ab962cfa9bcb4c4d50f3aea5406ea) - Make the identifier format strict.
* [3e1620a7](https://github.com/nihei9/vartan/commit/3e1620a781fe0eb097a9624cffb408bfb32bd5c8) - Add spelling inconsistencies check.
* [abf78681](https://github.com/nihei9/vartan/commit/abf78681e21afb3da1aad23060d3336dd8a7ee9f) - Follow golangci-lint.

[Changes](https://github.com/nihei9/vartan/compare/v0.4.1...v0.5.0)

## v0.4.1

* [18a3317](https://github.com/nihei9/vartan/commit/18a3317ac9c79651e5c74a2afc6b14fd9a3f9d4a), [97d3696](https://github.com/nihei9/vartan/commit/97d36965cbb30108340727a982539e67dafea92d), [8340b9f](https://github.com/nihei9/vartan/commit/8340b9f1dc1339d88762f361e284ea4ad6c079d7), [533c454](https://github.com/nihei9/vartan/commit/533c4545213b01d12a800c1c9d4ce2c85a12ae48) - Enhance tests.
* [389dd01](https://github.com/nihei9/vartan/commit/389dd0121475bdba7dea54f4cb02287fa48718da) -  Prohibit specifying associativity and precedence multiple times for a symbol.
* [9a9444b](https://github.com/nihei9/vartan/commit/9a9444bdc00e2a738fb0aa7cac4afa8a123d679b) - Prohibit using the same element multiple times in an `#ast` directive.
* [8bf4d23](https://github.com/nihei9/vartan/commit/8bf4d234d0b983d92378ba91660cae30e35f16f0) - Prohibit ambiguous symbol in an `#ast` directive.
* [b0bf8eb](https://github.com/nihei9/vartan/commit/b0bf8ebcc335b4193982b971e7779bd0d973421f) - Update dependencies.
* [0aa3e53](https://github.com/nihei9/vartan/commit/0aa3e53b50649052371edc9c09b470a63f889371) - `vartan show` command prints only adopted actions when conflicts occur.
* [0f5c301](https://github.com/nihei9/vartan/commit/0f5c30198eae1777262aaa6c65d8b59875049beb) - Suppress a report about conflicts resolved explicitly.

[Changes](https://github.com/nihei9/vartan/compare/v0.4.0...v0.4.1)

## v0.4.0

* [ed2c201](https://github.com/nihei9/vartan/commit/ed2c20102659f4c8aef0e88ea604e91fb56f25f6) - Change semantic action APIs. A parser reports whether it recovered from an error to the semantic action APIs via the argument `recovered`.
* [72da4b0](https://github.com/nihei9/vartan/commit/72da4b04e42baf3743ecf54b207f446a570d55e2) - Add `SemanticActionSet.TrapAndShiftError` method instead of `TrapError` and `ShiftError` methods.
* [0cf26ed](https://github.com/nihei9/vartan/commit/0cf26ed10f2563ea6721590ddbd5cccc7fa502b1) - Call `SemanticActionSet.MissError` method when an input doesn't meet an error production.
* [a57fda7](https://github.com/nihei9/vartan/commit/a57fda765cd32b44cd069da1c9a442b701b36dc2) - Pass a token that caused a syntax error to the semantic action APIs.
* [1dd7fef](https://github.com/nihei9/vartan/commit/1dd7fefb8c18da0c60807b14389cefbfdbc65993) - Generate the lexer source code.
* [83bc2b1](https://github.com/nihei9/vartan/commit/83bc2b1307d0e73424437649d26b804f20a83c38), [ba524fa](https://github.com/nihei9/vartan/commit/ba524fa0d49f597a4ace4bec72802334a0972c7a), [d3867e0](https://github.com/nihei9/vartan/commit/d3867e0769a90be422e2514e16017236e040a130), [d0431e3](https://github.com/nihei9/vartan/commit/d0431e3a435e2ad3180d945f66098c04ed0faf22) - Add `vartan-go` command.
* [5212b7f](https://github.com/nihei9/vartan/commit/5212b7fb22a762e81456134418bfe482a8704434), [bb9bf49](https://github.com/nihei9/vartan/commit/bb9bf495bd6cee65d8bc821939051d1be99861cc) - Use [golangci-lint](https://golangci-lint.run/).
* [1746609](https://github.com/nihei9/vartan/commit/1746609e248151d575f6e3913ad5023fd421bfff), [ed43562](https://github.com/nihei9/vartan/commit/ed43562cf58e8c0f9390421848879308fdfc60cb), [4d2a389](https://github.com/nihei9/vartan/commit/4d2a389c0ea605413d1cc89ae35f2a3aaa293072) - Use IDs and labels as parameters of an `#ast` directive instead of symbol positions.
* [dbd2e20](https://github.com/nihei9/vartan/commit/dbd2e20de97cdef56da0de07adff4251de94ef43) - Change syntax of production directives. The position of directives given to productions has moved from before a left-hand side value to after a left-hand side value.
This change aims to simplify the syntax.
* [90f28b5](https://github.com/nihei9/vartan/commit/90f28b5f7e7ef08e107e38002d122825764aad09) - Move all directives given to lexical productions from alternative directives to production directives.
This change aims to ensure consistency with respect to the syntax of definitions of terminal symbols and non-terminal symbols.
* [a1e4ae7](https://github.com/nihei9/vartan/commit/a1e4ae763cbf824f0d32a706cfe0d9603ce99b02) - Allow an alternative to have multiple directives.
* [1d0a67b](https://github.com/nihei9/vartan/commit/1d0a67bb7e95038f97e5a6c66bd2705d65f0ab57), [b565c7d](https://github.com/nihei9/vartan/commit/b565c7ddb7cbbf2ccfb8653c9a77140d83e02c55) - Update dependency.
* [5c26f61](https://github.com/nihei9/vartan/commit/5c26f617583463382978429f4c3fe550de521d42) - Print a parse tree in `vartan parse` command even if syntax errors occur.
When there is a parse tree, print it.
* [0636432](https://github.com/nihei9/vartan/commit/0636432f9051797b22e5c77722541c47edb312a0) - Remove `--grammar` option from `vartan compile` command.
* [8a6cfba](https://github.com/nihei9/vartan/commit/8a6cfbae9078c2095cb242e903dcac1c38c2fdb0), [8fda704](https://github.com/nihei9/vartan/commit/8fda704486c0bfbb9fead619b47f7ca987d56e4b), [180cac3](https://github.com/nihei9/vartan/commit/180cac37e53692c09763fc7bb49ac9ead44409ed) - Update documents.
* [14b2d7e](https://github.com/nihei9/vartan/commit/14b2d7e2728ab0314db56fc6e493d06fa285d006) - Allow arbitrary user-defined types for nodes in a syntax tree.

[Changes](https://github.com/nihei9/vartan/compare/v0.3.0...v0.4.0)

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
