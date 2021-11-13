package signature

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// LoadSignature load the signature file
func LoadSignature(file string) (*Signature, error) {
	var signature Signature
	meta, err := toml.DecodeFile(file, &signature)

	if !meta.IsDefined("version") {
		return nil, errors.New("version is not defined")
	}

	if signature.Version != "1" {
		return nil, errors.New("invalid version")
	}

	if !meta.IsDefined("signature") {
		return nil, errors.New("signature is not defined")
	}

	hash := sha1.New()
	for _, sig := range signature.Signatures {
		if sig.ID == "" {
			sig.ID = hex.EncodeToString(hash.Sum([]byte(sig.MatchString)))
			hash.Reset()
		}

		switch sig.Type {
		case extenstionType:
			sig.Match = sig.MatchString
		case pathType:
			sig.Match = sig.MatchString
		case filenameType:
			sig.Match = sig.MatchString
		case contentType:
			sig.Match = sig.MatchString
		case contentRegexType:
			sig.Match = regexp.MustCompile(sig.MatchString)
		}
	}

	return &signature, err
}

// ExtractMatch extract any match with signatures given filename and filecontent
func ExtractMatch(filename, content string, signatures []Base) []Match {
	var matches []Match

	for _, signature := range signatures {
		if !signature.Enable {
			continue
		}

		switch signature.Type {
		case extenstionType:
			baseFilename := filepath.Base(filename)
			needle := signature.Match.(string)
			if strings.HasPrefix(baseFilename, needle) {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Comment:     signature.Comment,
					Description: signature.Description,
					Substring:   baseFilename,
					Filename:    filename,
				})
			}
		case filenameType:
			baseFilename := filepath.Base(filename)
			needle := signature.Match.(string)
			if ok, _ := filepath.Match(needle, baseFilename); ok {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Comment:     signature.Comment,
					Description: signature.Description,
					Substring:   baseFilename,
					Filename:    filename,
				})
			}
		case pathType:
			needle := signature.Match.(string)
			if ok, _ := filepath.Match(needle, filename); ok {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Comment:     signature.Comment,
					Description: signature.Description,
					Substring:   filename,
					Filename:    filename,
				})
			}
		case contentType:
			needle := signature.Match.(string)
			if idx := strings.Index(content, needle); idx != -1 {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Comment:     signature.Comment,
					Description: signature.Description,
					Substring:   content[idx : idx+len(needle)],
					Filename:    filename,
				})
			}
		case contentRegexType:
			re := signature.Match.(*regexp.Regexp)
			match := re.FindString(content)
			if match != "" {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Comment:     signature.Comment,
					Description: signature.Description,
					Substring:   match,
					Filename:    filename,
				})
			}
		}
	}

	return matches
}
