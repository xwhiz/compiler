COMPILER_APP ?= cforge
COMPILER_MAIN ?= ./cmd/cforge
COMPILER_BIN ?= bin/$(COMPILER_APP)

RUNTIME_APP ?= crun
RUNTIME_MAIN ?= ./cmd/crun
RUNTIME_BIN ?= bin/$(RUNTIME_APP)

FILE ?= samples/slice9_run.c
OBJ ?= program.vmo

.PHONY: build build-compiler build-runtime test compile run-object run tokens ast sema ir codegen phases to_file plan runtime-note clean

build: build-compiler build-runtime

build-compiler:
	@mkdir -p bin
	@go build -o $(COMPILER_BIN) $(COMPILER_MAIN)

build-runtime:
	@mkdir -p bin
	@go build -o $(RUNTIME_BIN) $(RUNTIME_MAIN)

test:
	@go test ./...

compile: build-compiler
	@$(COMPILER_BIN) -o $(OBJ) $(FILE)

run-object: build-runtime
	@$(RUNTIME_BIN) $(OBJ)

run: build
	@$(COMPILER_BIN) -o $(OBJ) $(FILE)
	@$(RUNTIME_BIN) $(OBJ)

tokens: build-compiler
	@$(COMPILER_BIN) --tokens $(FILE)

ast: build-compiler
	@$(COMPILER_BIN) --ast $(FILE)

sema: build-compiler
	@$(COMPILER_BIN) --sema $(FILE)

ir: build-compiler
	@$(COMPILER_BIN) --ir $(FILE)

codegen: build-compiler
	@$(COMPILER_BIN) --codegen $(FILE)

phases: build
	@printf 'TOKENS\n'
	@$(COMPILER_BIN) --tokens $(FILE)
	@printf '\nAST\n'
	@$(COMPILER_BIN) --ast $(FILE)
	@printf '\nSEMANTIC ANALYSIS\n'
	@$(COMPILER_BIN) --sema $(FILE)
	@printf '\nIR\n'
	@$(COMPILER_BIN) --ir $(FILE)
	@printf '\nCODEGEN / OBJECT\n'
	@$(COMPILER_BIN) --codegen $(FILE)
	@printf '\nCOMPILE\n'
	@$(COMPILER_BIN) -o $(OBJ) $(FILE)
	@printf '\nRUNTIME OUTPUT\n'
	@$(RUNTIME_BIN) $(OBJ)

to_file: build
	@$(COMPILER_BIN) --tokens $(FILE) > 1_tokens.ansi
	@$(COMPILER_BIN) --ast $(FILE) > 2_ast.ansi
	@$(COMPILER_BIN) --sema $(FILE) > 3_sema.ansi
	@$(COMPILER_BIN) --ir $(FILE) > 4_ir.ansi
	@$(COMPILER_BIN) --codegen $(FILE) > 5_codegen.ansi
	@$(COMPILER_BIN) -o $(OBJ) $(FILE) >/dev/null
	@$(RUNTIME_BIN) $(OBJ) > 6_output.ansi

plan:
	@grep -n "Slice 11\|Slice 12\|cforge\|crun" PLAN.md || true

runtime-note:
	@printf 'Compiler binary: %s\n' $(COMPILER_BIN)
	@printf 'Runtime binary: %s\n' $(RUNTIME_BIN)
	@printf 'Source file: %s\n' $(FILE)
	@printf 'Object file: %s\n' $(OBJ)

clean:
	@rm -f $(COMPILER_BIN) $(RUNTIME_BIN)
