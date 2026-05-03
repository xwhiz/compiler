# mycc

Handwritten Go compiler for C-like subset. Pipeline:

1. lexer
2. parser / AST
3. semantic analysis
4. IR
5. VM codegen
6. VM execution

## Build

```bash
go build ./...
```

Run compiler:

```bash
go run ./cmd/mycc <file.c>
```

## Phase Flags

```bash
go run ./cmd/mycc --tokens <file.c>
go run ./cmd/mycc --ast <file.c>
go run ./cmd/mycc --sema <file.c>
go run ./cmd/mycc --ir <file.c>
go run ./cmd/mycc --codegen <file.c>
```

Without phase flag, compiler builds VM program and runs it.

## Supported Features

### Types

- `int`
- `char`
- `float`
- `void`

### Statements

- block
- declaration
- assignment
- expression statement
- `if` / `else`
- `while`
- `return`

### Expressions

- int / float / char / string literals
- identifiers
- 1D array indexing
- function calls
- unary `-`, `!`
- binary `+ - * / % < <= > >= == != && ||`

### Functions

- user-defined functions
- parameters
- return values
- function must be defined before call

### Arrays

- 1D fixed-size arrays only
- `int`, `char`, `float` arrays
- `char s[N] = "literal"`

### Builtins

- `print_int(int)`
- `print_float(float)`
- `print_char(char)`
- `print_str(char[])`
- `print_newline()`

## Sample Programs

Main slice samples:

- `samples/slice7_run.c`
- `samples/slice8_run.c`
- `samples/slice9_run.c`
- `samples/slice6_run.c`
- `samples/slice5_run.c`

Passing samples:

- `samples/pass/function_add.c`
- `samples/pass/function_chain.c`
- `samples/pass/char_string.c`
- `samples/pass/arrays_basic.c`
- `samples/pass/float_math.c`
- `samples/pass/if_else.c`
- `samples/pass/while_sum.c`
- `samples/pass/scope_shadow.c`

Failing samples:

- `samples/fail/call_before_def.c`
- `samples/fail/wrong_arg_count.c`
- `samples/fail/duplicate_function.c`
- `samples/fail/duplicate_decl.c`
- `samples/fail/missing_semicolon.c`
- `samples/fail/undeclared_var.c`
- `samples/fail/string_too_long.c`
- `samples/fail/array_init_unsupported.c`
- `samples/fail/float_to_int.c`
- `samples/fail/array_as_scalar.c`

## Current Limits

- no `for`
- no 2D arrays
- no array params
- no optimizer pass yet
- no escape sequences in char/string literals yet
- logical `&&` and `||` evaluate both sides
- no global variables yet
