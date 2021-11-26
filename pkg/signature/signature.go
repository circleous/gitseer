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
	for i := range signature.Signatures {
		if signature.Signatures[i].ID == "" {
			signature.Signatures[i].ID = hex.EncodeToString(hash.Sum([]byte(signature.Signatures[i].MatchString)))
			hash.Reset()
		}

		switch signature.Signatures[i].Type {
		case extenstionType:
			signature.Signatures[i].Match = signature.Signatures[i].MatchString
		case pathType:
			signature.Signatures[i].Match = regexp.MustCompile(signature.Signatures[i].MatchString)
		case filenameType:
			signature.Signatures[i].Match = signature.Signatures[i].MatchString
		case contentType:
			signature.Signatures[i].Match = regexp.MustCompile(signature.Signatures[i].MatchString)
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
			if strings.HasSuffix(baseFilename, needle) {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Description: signature.Description,
					Substring:   baseFilename,
				})
			}
		case filenameType:
			baseFilename := filepath.Base(filename)
			needle := signature.Match.(string)
			if ok, _ := filepath.Match(needle, baseFilename); ok {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Description: signature.Description,
					Substring:   baseFilename,
				})
			}
		case pathType:
			re := signature.Match.(*regexp.Regexp)
			if re.FindString(filename) != "" {
				matches = append(matches, Match{
					SignatureID: signature.ID,
					Description: signature.Description,
					Substring:   filename,
				})
			}
		case contentType:
			re := signature.Match.(*regexp.Regexp)
			founds := re.FindAllStringIndex(content, -1)
			if founds != nil {
				for _, found := range founds {
					matches = append(matches, Match{
						SignatureID: signature.ID,
						Description: signature.Description,
						Substring:   content[found[0]:found[1]],
						LineNumber:  StringPosToLineNumber(content, found[0]),
					})
				}
			}

		}
	}

	return matches
}
