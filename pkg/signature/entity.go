package signature

const (
	extenstionType = "extension"
	filenameType   = "filename"
	pathType       = "path"
	contentType    = "content"
)

// Metadata contains the version and signature author
type Metadata struct {
	// Version
	Version string `toml:"version"`
}

// Base is the struct for signature data
type Base struct {
	// Type is the signature type [ extension filename path content ]
	Type string `toml:"type"`

	// Unique ID for the signature can be set manually or if empty, program will
	// assign a SHA-1 hash of the match string
	ID string `toml:"id"`

	// Entropy
	Entropy float64 `toml:"entropy"`

	// Description of the signature, added to findings
	Description string `toml:"description"`

	// Enable if set to false, signature will not be used to extract match
	Enable bool `toml:"enable"`

	// MatchString will be converted to Match by signature type
	MatchString string `toml:"match"`
	// Match is the "compiled" version of the MatchString
	Match interface{}
}

// Signature is the signature struct used for loading from file
type Signature struct {
	// Metadata signature metadata
	Metadata
	// Signatures contains slices of signatures
	Signatures []Base `toml:"signature"`
}

// Match is the struct used for holding the data when there's a match in
// ExtractMtach
type Match struct {
	Substring   string
	SignatureID string
	Description string
	LineNumber  int32
}
