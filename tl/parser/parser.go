package parser

import (
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
)

// Parser parses TL tokens into an AST Schema.
type Parser struct {
	lexer      *Lexer
	cur        Token
	isFunction bool
}

// Parse parses a raw TL schema string into a Schema.
func Parse(input string) (*Schema, error) {
	p := &Parser{
		lexer: NewLexer(input),
	}
	p.nextToken()

	schema := &Schema{}

	for p.cur.Type != TokenEOF {
		switch p.cur.Type {
		case TokenSectionTypes:
			p.isFunction = false
			p.nextToken()
		case TokenSectionFunctions:
			p.isFunction = true
			p.nextToken()
		case TokenSemicolon:
			p.nextToken()
		default:
			def, err := p.parseDefinition()
			if err != nil {
				return nil, err
			}
			if def != nil {
				if def.IsFunction {
					schema.Functions = append(schema.Functions, *def)
				} else {
					schema.Constructors = append(schema.Constructors, *def)
				}
			}
		}
	}

	return schema, nil
}

func (p *Parser) nextToken() {
	p.cur = p.lexer.NextToken()
}

func (p *Parser) parseDefinition() (*Definition, error) {
	// Skip curly brace generic params if any: {X:Type}
	if p.cur.Type == TokenBraceOpen {
		for p.cur.Type != TokenBraceClose && p.cur.Type != TokenEOF {
			p.nextToken()
		}
		if p.cur.Type == TokenBraceClose {
			p.nextToken()
		}
	}

	if p.cur.Type != TokenIdent {
		return nil, fmt.Errorf("line %d: expected combinator name, got %v (%q)", p.cur.Line, p.cur.Type, p.cur.Value)
	}

	name := p.cur.Value
	p.nextToken()

	var id uint32
	var hasExplicitID bool

	// Check for #hexID
	if p.cur.Type == TokenHexID {
		parsed, err := strconv.ParseUint(p.cur.Value, 16, 32)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid hex ID %q: %w", p.cur.Line, p.cur.Value, err)
		}
		id = uint32(parsed)
		hasExplicitID = true
		p.nextToken()
	}

	var params []Param
	var sigParts []string
	sigParts = append(sigParts, name)

	// Parse parameters until '='
	for p.cur.Type != TokenEquals && p.cur.Type != TokenEOF && p.cur.Type != TokenSemicolon {
		if p.cur.Type == TokenBraceOpen {
			// Skip inline generics: {t:Type}
			for p.cur.Type != TokenBraceClose && p.cur.Type != TokenEOF {
				p.nextToken()
			}
			if p.cur.Type == TokenBraceClose {
				p.nextToken()
			}
			continue
		}

		if p.cur.Type != TokenIdent {
			p.nextToken()
			continue
		}

		paramName := p.cur.Value
		p.nextToken()

		if p.cur.Type != TokenColon {
			continue
		}
		p.nextToken() // consume ':'

		param, sigPart, err := p.parseParamType(paramName)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
		sigParts = append(sigParts, sigPart)
	}

	if p.cur.Type != TokenEquals {
		return nil, fmt.Errorf("line %d: expected '=' in declaration %s", p.cur.Line, name)
	}
	p.nextToken() // consume '='

	// Parse result type (e.g. InputFile, Vector<Updates>, etc.)
	resultType, err := p.parseResultType()
	if err != nil {
		return nil, err
	}

	// Consume terminating ';'
	if p.cur.Type == TokenSemicolon {
		p.nextToken()
	}

	// If no explicit #hex was in the definition, calculate CRC32 of normalized representation
	if !hasExplicitID {
		fullSig := strings.Join(sigParts, " ") + " = " + resultType
		id = crc32.ChecksumIEEE([]byte(fullSig))
	}

	return &Definition{
		Name:       name,
		ID:         id,
		Params:     params,
		ResultType: resultType,
		IsFunction: p.isFunction,
	}, nil
}

func (p *Parser) parseParamType(paramName string) (Param, string, error) {
	param := Param{Name: paramName}

	rawType := p.cur.Value
	p.nextToken()

	// Check if this is a conditional flag: flags.0?type or flags.3?true
	if p.cur.Type == TokenQuestion {
		p.nextToken() // consume '?'
		condType := p.cur.Value
		p.nextToken() // consume condType

		// Check if condType is Vector<T>
		if condType == "Vector" && p.cur.Type == TokenAngleOpen {
			p.nextToken()
			elem := p.cur.Value
			p.nextToken()
			if p.cur.Type == TokenAngleClose {
				p.nextToken()
			}
			condType = "Vector<" + elem + ">"
			param.IsVector = true
			param.ElemType = elem
		}

		// rawType was "flags.0"
		dotIdx := strings.IndexByte(rawType, '.')
		flagField := "flags"
		bit := 0
		if dotIdx > 0 {
			flagField = rawType[:dotIdx]
			if parsedBit, err := strconv.Atoi(rawType[dotIdx+1:]); err == nil {
				bit = parsedBit
			}
		}

		param.Type = condType
		param.Flag = &FlagCondition{
			Field:  flagField,
			Bit:    bit,
			IsTrue: condType == "true",
		}
		sigPart := fmt.Sprintf("%s:%s?%s", paramName, rawType, condType)
		return param, sigPart, nil
	}

	// Check if rawType is Vector<T>
	if rawType == "Vector" && p.cur.Type == TokenAngleOpen {
		p.nextToken() // consume '<'
		elem := p.cur.Value
		p.nextToken() // consume elem
		if p.cur.Type == TokenAngleClose {
			p.nextToken() // consume '>'
		}
		param.Type = "Vector<" + elem + ">"
		param.IsVector = true
		param.ElemType = elem
		sigPart := fmt.Sprintf("%s:Vector<%s>", paramName, elem)
		return param, sigPart, nil
	}

	param.Type = rawType
	sigPart := fmt.Sprintf("%s:%s", paramName, rawType)
	return param, sigPart, nil
}

func (p *Parser) parseResultType() (string, error) {
	if p.cur.Type != TokenIdent {
		return "", fmt.Errorf("line %d: expected result type, got %v", p.cur.Line, p.cur.Type)
	}
	result := p.cur.Value
	p.nextToken()

	// Handle generic result types like Vector<T>
	if result == "Vector" && p.cur.Type == TokenAngleOpen {
		p.nextToken()
		elem := p.cur.Value
		p.nextToken()
		if p.cur.Type == TokenAngleClose {
			p.nextToken()
		}
		result = "Vector<" + elem + ">"
	}

	return result, nil
}
