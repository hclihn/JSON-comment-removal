package main

import (
  "fmt"
  "bytes"
  "unicode/utf8"
)

func WrapTraceableErrorf(e error, f string, extra ...interface{}) error {
  if e != nil {
    return fmt.Errorf(f+": %w", append(extra, e)...)
  }
  return fmt.Errorf(f, extra...)
}

type jsonCmtScannerState int

const (
  jsonNormal jsonCmtScannerState = iota
  jsonInQuote
  jsonEscape
  jsonTestComment
  jsonLineComment
  jsonBlockComment
  jsonTestEndBlkCmt
)

func RemoveJSONComment(js []byte, inPlace bool) ([]byte, error){
  l, sz := len(js), 0
  state := jsonNormal
  var b bytes.Buffer
  var r rune
  for i := 0; i < l; i += sz {
    r, sz = utf8.DecodeRune(js[i:])
    if r == utf8.RuneError { // encoding error
			return nil, WrapTraceableErrorf(nil,
				"failed to remove JSON comment: invalid Unicode encoding char '%c' at index %d", js[i], i)
		}
    switch state {
    case jsonNormal:
      switch r {
      case '/':
        state = jsonTestComment
        continue
      case '"': // start of quote
        state = jsonInQuote
      }
      if _, err := b.WriteRune(r); err != nil {
        return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '%c' to buffer", r)
      }
    case jsonInQuote:
      switch r {
      case '\\': // escape code
        state = jsonEscape
      case '"': // end of quote
        state = jsonNormal
      }
      if _, err := b.WriteRune(r); err != nil {
        return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '%c' to buffer", r)
      }
    case jsonEscape:
      if _, err := b.WriteRune(r); err != nil {
        return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '%c' to buffer", r)
      }
      state = jsonInQuote
    case jsonTestComment: // test char following '/'
      switch r {
      case '/':
        state = jsonLineComment
        continue
      case '*':
        state = jsonBlockComment
        continue
      case '"': // unlikely, but we will leave it to JSON parser
        state = jsonInQuote
      default: // unlikely, but we will leave it to JSON parser
        state = jsonNormal
      }
      if _, err := b.WriteRune('/'); err != nil {
        return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '/' to buffer")
      }
      if _, err := b.WriteRune(r); err != nil {
        return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '%c' to buffer", r)
      }
    case jsonLineComment:
      if r == '\n' {
        if _, err := b.WriteRune(r); err != nil {
          return nil, WrapTraceableErrorf(err, "failed to remove JSON comment when writing rune '%c' to buffer", r)
        }
        state = jsonNormal
      }
    case jsonBlockComment:
      if r == '*' {
        state = jsonTestEndBlkCmt
      }
    case jsonTestEndBlkCmt:
      if r == '/' {
        state = jsonNormal
      } else if r != '*' {
        state = jsonBlockComment
      }
    default:
      return nil, WrapTraceableErrorf(nil, "failed to remove JSON comment: unknown JSON commnet scanner state %d", state)
    }
  }
  if inPlace && len(js) > b.Len() {
    n, err := b.Read(js)
    if err != nil {
      return nil, WrapTraceableErrorf(err, "failed to remove JSON comment in place")
    }
    for i := n; i < len(js); i++ {
      js[i] = 0x20
    }
    return nil, nil
  }
  return b.Bytes(), nil
}


func main() {
  js := `{
  "FormatVersion": "discovery:1", 
  "Description": "// test this", // Manifest description. Required. 
  "LastChanged": "2020-07-14T22:19:59Z", /* Last modified time. Format: 
  YYYY-MM-DDThh:mm:ssZ. Required. */
  "ChangedBy": "/* and this */", /* Last changed by (person). Required. */
  "WardenPath": "", /* Warden JSON path to the base data to be used in this manifest. */
  "MbManifestPath": "\"this\"/* test */", /* Template string evaluated to Motherboard Manifest path under manifest/. */
  "MbManifestName": "", /* Template string evaluated to Motherboard 
  
  Manifest name after its prefix. Required. */
  "Rules": [{
    "Description": "\"and\"// test", /* Description of the rule. Required. */
    "Condition": "", /* Template string evaluated to a result of a JSONBoolExpressionString. When false, the rule is skipped. An empty one is always true. */
    "Config": "" /* Template string to formulate the full config code. Required. */
  }], /* Rules to discover the full config code. Required. */
  "Globals": [{
    "Name": "", /* String as the global variable name. Required. */
    "Value": "" /* Template string evaluated to the variable's string value. Required. */
  }] 
}`
  inPlace := true
  bb := []byte(js)
  b, e := RemoveJSONComment(bb, inPlace)
  if e != nil {
    fmt.Printf("Failed to remove JSON comment: %s\n", e)
    return 
  }
  if inPlace {
    fmt.Printf("JSON comments removed:\n%s\n", string(bb))
  } else {
    fmt.Printf("JSON comments removed:\n%s\n", string(b))
  }
}