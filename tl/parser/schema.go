package parser

// Schema represents a parsed Telegram Type Language (TL) schema.
type Schema struct {
	Constructors []Definition
	Functions    []Definition
}

// Definition represents a single TL constructor or function combinator.
type Definition struct {
	Name       string  // e.g. "auth.sentCode", "upload.saveFilePart"
	ID         uint32  // e.g. 0xb304a621
	Params     []Param // list of parameters
	ResultType string  // e.g. "auth.SentCode", "Bool"
	IsFunction bool    // true if defined in ---functions--- section
}

// Param represents a single field or argument in a TL combinator.
type Param struct {
	Name      string         // field name, e.g. "file_id", "flags"
	Type      string         // type name, e.g. "int", "long", "string", "bytes", "Vector<T>"
	Flag      *FlagCondition // non-nil if this field is conditional on a bitmask flag
	IsVector  bool           // true if Vector<T>
	ElemType  string         // element type if IsVector
}

// FlagCondition represents a conditional field governed by a bit in a flags field.
// Example: flags.0?true or flags.2?InputFile
type FlagCondition struct {
	Field  string // the flag field name, e.g. "flags"
	Bit    int    // bit index (0..31)
	IsTrue bool   // true if conditional type is "true" (a presence indicator rather than data)
}
