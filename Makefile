APP = mycc
MAIN = ./cmd/mycc
BIN = $(APP)

FILE ?= input.txt

all: build
	@echo "TOKENS:"
	@./$(BIN) --tokens $(FILE)
	@echo "============================================="
	
	@echo "AST:"
	@./$(BIN) --ast $(FILE)
	@echo "============================================="
	
	@echo "Semantic Analysis:"
	@./$(BIN) --sema $(FILE)
	@echo "============================================="
	
	@echo "IR:"
	@./$(BIN) --ir $(FILE)
	@echo "============================================="
	
	@echo "Codegen:"
	@./$(BIN) --codegen $(FILE)
	@echo "============================================="
	
	@echo "Executed:"
	@./$(BIN) $(FILE)

to_file: build
	@echo "TOKENS:"
	@./$(BIN) --tokens $(FILE) > 1_tokens.ansi
	@echo "============================================="
	
	@echo "AST:"
	@./$(BIN) --ast $(FILE) > 2_ast.ansi
	@echo "============================================="
	
	@echo "Semantic Analysis:"
	@./$(BIN) --sema $(FILE) > 3_sema.ansi
	@echo "============================================="
	
	@echo "IR:"
	@./$(BIN) --ir $(FILE) > 4_ir.ansi
	@echo "============================================="
	
	@echo "Codegen:"
	@./$(BIN) --codegen $(FILE) > 5_codegen.ansi
	@echo "============================================="
	
	@echo "Executed:"
	@./$(BIN) $(FILE) > 6_output.ansi

build:
	@go build -o $(BIN) $(MAIN)

test: build
	@go test ./...