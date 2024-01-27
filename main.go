package main

import (
	"fmt"
	"os"
  "runtime/debug"
)

// https://www.json.org/json-en.html

// This implementation sets no limits on nesting depths
// https://www.rfc-editor.org/rfc/rfc8259.html#section-9

var singleChars = map[rune]bool{
  '{': true,
  '}': true,
  '[': true,
  ']': true,
  ',': true,
  ':': true,
}
var wsChars = map[rune]bool{
  ' ': true,
  '\t': true,
  '\n': true,
  '\r': true,
}
var keywords = map[string]bool{
  "true": true,
  "false": true,
  "null": true,
}
var escapes = map[rune]bool{
  '\\': true,
  '"': true,
  '/': true,
  'b': true,
  'f': true,
  'n': true,
  'r': true,
  't': true,
  'u': true,
}

func tokenize(input string) ([]string, error) {
  var tokens []string
  var currentToken string
  inString := false
  inEscape := false
  for _, char := range input {
    startString := !inString && char == '"'
    endString := inString && char == '"' && !inEscape
    if startString || endString {
      if !inString {
        inString = true
        currentToken = "\""
      } else {
        inString = false
        currentToken += "\""
        tokens = append(tokens, currentToken)
        currentToken = ""
      }
      continue
    }
    if inString {
      if inEscape {
        if _, ok := escapes[char]; !ok {
          return nil, fmt.Errorf("invalid escape char: %c", char)
        }
        inEscape = false
      } else if char == '\\' && !inEscape {
        inEscape = true
      }
      currentToken += string(char)
      continue
    }
    if _, ok := singleChars[char]; ok {
      if len(currentToken) > 0 {
        tokens = append(tokens, currentToken)
        currentToken = ""
      }
      tokens = append(tokens, string(char))
      continue
    }
    if _, ok := wsChars[char]; ok {
      continue
    }
    // Other values - 'true', 'false', 'null', numbers
    currentToken += string(char)
  }
  return tokens, nil
}

// json
//   element
func parse(tokens []string) error {
  if len(tokens) == 0 {
    return fmt.Errorf("empty input")
  }
  var err error
  idx := 0
  if tokens[0] == "{" {
    idx, err = parseObject(idx, tokens)
    if err != nil {
      return fmt.Errorf("parseObject(): %w", err)
    }
  } else if tokens[0] == "[" {
    idx, err = parseArray(idx, tokens)
    if err != nil {
      return fmt.Errorf("parseArray(): %w", err)
    }
  } else {
    return fmt.Errorf("JSON payload should be object or array")
  }
  if idx != len(tokens) {
    return fmt.Errorf("unexpected token: %s", tokens[idx])
  }
  return nil
}

// Accessing token within tokens
func tokenInBounds(index int, tokens []string) bool {
  return index >= 0 && index < len(tokens)
}
func getToken(index int, tokens []string) (string, error) {
  if !tokenInBounds(index, tokens) {
    debug.PrintStack()
    return "", fmt.Errorf("token index out of range")
  }
  return tokens[index], nil
}
// Accessing runes within a token
func runeInBounds(index int, token string) bool {
  return index >= 0 && index < len(token)
}
func getRune(index int, token string) (rune, error) {
  if !runeInBounds(index, token) {
    debug.PrintStack()
    return 0, fmt.Errorf("rune index %d out of range in %s", index, token)
  }
  return []rune(token)[index], nil
}

// element
//   ws value ws
func parseElement(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if isWS(token) {
    currentTokenIdx++
  }
  currentTokenIdx, err = parseValue(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseValue(): %w", err)
  }
  if !tokenInBounds(currentTokenIdx, tokens) {
    return currentTokenIdx, nil
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if isWS(token) {
    currentTokenIdx++
  }
  return currentTokenIdx, nil
}

// elements
//   element
//   element ',' elements
func parseElements(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  currentTokenIdx, err = parseElement(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseElement(): %w", err)
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token == "," {
    return parseElements(currentTokenIdx+1, tokens)
  }
  return currentTokenIdx, nil
}

// value
//   object
//   array
//   string
//   number
//   "true"
//   "false"
//   "null"
func parseValue(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, err
  }
  if token == "{" {
    return parseObject(currentTokenIdx, tokens)
  }
  if token == "[" {
    return parseArray(currentTokenIdx, tokens)
  }
  if token[0] == '"' {
    return parseString(currentTokenIdx, tokens)
  }
  if _, ok := keywords[token]; ok {
    return currentTokenIdx+1, nil
  }
  if _, err := parseNumber(currentTokenIdx, tokens); err != nil {
    return currentTokenIdx, fmt.Errorf("parseNumber(): %w", err)
  }
  return currentTokenIdx+1, nil
}

// number
//   integer fraction exponent
func parseNumber(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  idx := 0
  idx, err = parseInteger(idx, token)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseInteger(): %w", err)
  }
  if idx == len(token) {
    return currentTokenIdx+1, nil
  }
  inBounds := runeInBounds(idx, token)
  if !inBounds {
    return currentTokenIdx, fmt.Errorf("rune idx out of bounds")
  }
  c, err := getRune(idx, token)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getRune(): %w", err)
  }
  if c == '.' {
    idx, err = parseFraction(idx, token)
    if err != nil {
      return idx, fmt.Errorf("parseFraction(): %w", err)
    }
  }
  if idx == len(token) {
    return currentTokenIdx+1, nil
  }
  inBounds = runeInBounds(idx, token)
  if !inBounds {
    return currentTokenIdx, fmt.Errorf("rune idx out of bounds")
  }
  c, err = getRune(idx, token)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getRune(): %w", err)
  }
  if c == 'e' || c == 'E' {
    idx, err = parseExponent(idx, token)
    if err != nil {
      return idx, fmt.Errorf("parseExponent(): %w", err)
    }
  }
  if idx != len(token) {
    return idx, fmt.Errorf("Unexpected token: %s", token[idx:])
  }
  return currentTokenIdx+1, nil
}

// integer
//   digit
//   onenine digits
//   '-' digit
//   '-' onenine digits
func parseInteger(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c == '-' {
    idx++
    c, err = getRune(idx, token)
    if err != nil {
      return idx, fmt.Errorf("getRune(): %w", err)
    }
  }
  // onenine first case
  if c >= '1' && c <= '9' {
    idx, err := parseOnenine(idx, token)
    if err != nil {
      return idx, fmt.Errorf("parseOnenine(): %w", err)
    }
    if idx == len(token) {
      return idx, nil
    }
    idx, err = parseDigits(idx, token)
    if err != nil {
      return idx, fmt.Errorf("parseDigits(): %w", err)
    }
    return idx, nil
  }
  // digit first case
  idx, err = parseDigit(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigit(): %w", err)
  }
  return idx, nil
}

// digit
//   '0'
//    onenine
func parseDigit(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c == '0' {
    return idx+1, nil
  }
  idx, err = parseOnenine(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseOnenine(): %w", err)
  }
  return idx, nil
}

// digits
//   digit
//   digit digits
func parseDigits(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  // Stop at fraction or exponent
  if c == '.' || c == 'e' || c == 'E' {
    return idx, nil
  }
  idx, err = parseDigit(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigit(): %w", err)
  }
  if idx == len(token) {
    return idx, nil
  }
  idx, err = parseDigits(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigits(): %w", err)
  }
  return idx, nil
}

// onenine
//   '1' . '9'
func parseOnenine(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c < '1' || c > '9' {
    return idx, fmt.Errorf("Expected onenine, got %c in %s", c, token)
  }
  return idx+1, nil
}

// fraction
//   "." digits
func parseFraction(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c != '.' {
    return idx, fmt.Errorf("Expected '.', got %c in %s", c, token)
  }
  idx++
  idx, err = parseDigits(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigits(): %w", err)
  }
  return idx, nil
}

// exponent
//   'E' sign digits
//   'e' sign digits
func parseExponent(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c != 'E' && c != 'e' {
    return idx, fmt.Errorf("Expected 'E' or 'e', got %c in %s", c, token)
  }
  idx++
  c, err = getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c == '+' || c == '-' {
    idx++
  }
  idx, err = parseDigits(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigits(): %w", err)
  }
  return idx, nil
}

// sign
//   '+'
//   '-'
func parseSign(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c != '+' && c != '-' {
    return idx, fmt.Errorf("Expected '+', '-', got %c in %s", c, token)
  }
  return idx+1, nil
}

// members
//   member
//   member ',' members
func parseMembers(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  currentTokenIdx, err = parseMember(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseMember(): %w", err)
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token == string(',') {
    return parseMembers(currentTokenIdx+1, tokens)
  }
  return currentTokenIdx, nil
}

// member
//   ws string ws ':' element
func parseMember(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if isWS(token) {
    currentTokenIdx++
  }
  currentTokenIdx, err = parseString(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseString(): %w", err)
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if isWS(token) {
    currentTokenIdx++
    token, err = getToken(currentTokenIdx, tokens)
    if err != nil {
      return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
    }
  }
  if token != string(':') {
    return currentTokenIdx, fmt.Errorf("Expected ':', got %s", token)
  }
  currentTokenIdx++
  return parseElement(currentTokenIdx, tokens)  
}  

// string
//   '"' characters '"'
func parseString(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token[0] != '"' {
    debug.PrintStack()
    return currentTokenIdx, fmt.Errorf("expected string starting with \", got %s", token)
  }
  if token[len(token)-1] != '"' {
    return currentTokenIdx, fmt.Errorf("expected string ending with \", got %s", token)
  }
  idx := 1
  idx, err = parseCharacters(idx, token)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseCharacters(): %w", err)
  }
  return currentTokenIdx+1, nil
}

// characters
//   ""
//   character characters
func parseCharacters(idx int, token string) (int, error) {
  if idx == len(token)-1 {
    if token[idx] != '"' {
      return idx, fmt.Errorf("expected string ending with \", got %s", token)
    }
    return idx+1, nil
  }
  idx, err := parseCharacter(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseCharacter(): %w", err)
  }
  idx, err = parseCharacters(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseCharacters(): %w", err)
  }
  return idx, nil
}

// character
//   '0020' . '10FFFF' - '"' - '\'
//   '\' escape
func parseCharacter(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c == '\\' {
    return parseEscape(idx+1, token)
  }
  if c < 0x0020 || c > 0x10FFFF {
    return idx, fmt.Errorf("expected character, got %q, in %s", c, token)
  }
  return idx+1, nil
}

// escape
//   '"'
//   '\'
//   '/'
//   'b'
//   'f'
//   'n'
//   'r'
//   't'
//   'u' hex hex hex hex
func parseEscape(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  switch c {
    case 'u':
      idx, err = parseHex(idx+1, token)
      if err != nil {
        return idx, fmt.Errorf("parseHex(): %w", err)
      }
      idx, err = parseHex(idx, token)
      if err != nil {
        return idx, fmt.Errorf("parseHex(): %w", err)
      }
      idx, err = parseHex(idx, token)
      if err != nil {
        return idx, fmt.Errorf("parseHex(): %w", err)
      }
      idx, err = parseHex(idx, token)
      if err != nil {
        return idx, fmt.Errorf("parseHex(): %w", err)
      }
      return idx, nil
    case '"':
      return idx+1, nil
    case '\\':
      return idx+1, nil
    case '/':
      return idx+1, nil
    case 'b':
      return idx+1, nil
    case 'f':
      return idx+1, nil
    case 'n':
      return idx+1, nil
    case 'r':
      return idx+1, nil
    case 't':
      return idx+1, nil
  }
  return idx, fmt.Errorf("expected escape character, got %c in %s", c, token)
}

// hex
//   digit
//   'A' . 'F'
//   'a' . 'f'
func parseHex(idx int, token string) (int, error) {
  c, err := getRune(idx, token)
  if err != nil {
    return idx, fmt.Errorf("getRune(): %w", err)
  }
  if c >= 'A' && c <= 'F' {
    return idx+1, nil
  }
  if c >= 'a' && c <= 'f' {
    return idx+1, nil
  }
  idx, err = parseDigit(idx, token)
  if err != nil {
    return idx, fmt.Errorf("parseDigit(): %w", err)
  }
  return idx, nil
}

// object
//  '{' ws '}'
//  '{' members '}'
func parseObject(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token != "{" {
    return currentTokenIdx, fmt.Errorf("expected '{', got %s", token)
  }
  currentTokenIdx++
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  // empty object case
  if isWS(token) {
    currentTokenIdx++
    token, err = getToken(currentTokenIdx, tokens)
    if err != nil {
      return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
    }
  }
  if token == "}" {
    return currentTokenIdx+1, nil
  }
  currentTokenIdx, err = parseMembers(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseMembers(): %w", err)
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token == "}" {
    return currentTokenIdx+1, nil
  }
  return currentTokenIdx, fmt.Errorf("expected '}' but got '%s'", token)
}

// array
//   '[' ws ']'
//   '[' elements ']'
func parseArray(currentTokenIdx int, tokens []string) (int, error) {
  token, err := getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token != "[" {
    return currentTokenIdx, fmt.Errorf("expected '[' but got '%s'", token)
  }
  currentTokenIdx++
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  // empty object case
  if isWS(token) {
    currentTokenIdx++
    token, err = getToken(currentTokenIdx, tokens)
    if err != nil {
      return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
    }
  }
  if token == "]" {
    return currentTokenIdx+1, nil
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  currentTokenIdx, err = parseElements(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("parseElements(): %w", err)
  }
  token, err = getToken(currentTokenIdx, tokens)
  if err != nil {
    return currentTokenIdx, fmt.Errorf("getToken(): %w", err)
  }
  if token != "]" {
    return currentTokenIdx, fmt.Errorf("expected ']' but got '%s'", token)
  }
  return currentTokenIdx+1, nil
}

func isWS(token string) bool {
  for _, char := range token {
    if _, ok := wsChars[char]; !ok {
      return false
    }
  }
  return true
}

func main() {
	jsonFilename := os.Args[1]
  jsonFile, err := os.Open(jsonFilename)
  if err != nil {
    fmt.Println("error opening json file: ", err)
    os.Exit(1)
  }
  defer jsonFile.Close()

  jsonData, err := os.ReadFile(jsonFilename)
  if err != nil {
    fmt.Println("error reading json file: ", err)
    os.Exit(1)
  }

  tokens, err := tokenize(string(jsonData))
  if err != nil {
    fmt.Println("error tokenizing json file: ", err)
    os.Exit(1)
  }
  fmt.Println(tokens)
  for idx, token := range tokens {
    fmt.Printf("%d %s\n", idx, token)
  }
  if err = parse(tokens); err != nil {
    fmt.Println("error parsing json: ", err)
    os.Exit(1)
  }
}
