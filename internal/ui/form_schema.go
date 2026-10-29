package ui

// fieldKind identifies the type of form control used for a field.
type fieldKind int

const (
	fieldText     fieldKind = iota // text input (press i to edit)
	fieldNumber                    // numeric text input
	fieldPicker                    // option picker (enter/space opens dropdown)
	fieldSubModal                  // value shown inline; enter/space opens a sub-modal
)

// fieldOption is a selectable value for picker fields.
type fieldOption struct {
	Value   string // stored value ("" represents default/inherit)
	Display string // display text shown to the user
}

// fieldDef declaratively defines a single form field.
type fieldDef struct {
	Key         string             // unique identifier (maps to a config field)
	Label       string             // display label (without trailing colon)
	Kind        fieldKind          // control type
	Options     []fieldOption      // for Picker: available choices
	Default     string             // default value
	Placeholder string             // placeholder for text/number inputs
	Helper      string             // help text shown when field is focused
	CharLimit   int                // max input chars (0 = use default 256)
	Narrow      bool               // use narrow field width (e.g. Port)
	Validate    func(string) error // optional per-field validation

	// DisabledWhen returns a non-empty message when the field should be
	// rendered as disabled (e.g. "overridden by theme"). nil = always enabled.
	DisabledWhen func(values map[string]string) string
}

// isTextField returns true when the field uses a textinput.Model.
func (f *fieldDef) isTextField() bool {
	return f.Kind == fieldText || f.Kind == fieldNumber
}

// sectionDef groups related fields under a named section header.
type sectionDef struct {
	Key    string     // unique identifier
	Label  string     // display label
	Fields []fieldDef // ordered fields within the section
}

// formSchema is the complete declarative definition of a form's structure.
type formSchema struct {
	Sections []sectionDef
}

// fieldByKey returns the field definition for the given key, or nil.
func (s *formSchema) fieldByKey(key string) *fieldDef {
	for si := range s.Sections {
		for fi := range s.Sections[si].Fields {
			if s.Sections[si].Fields[fi].Key == key {
				return &s.Sections[si].Fields[fi]
			}
		}
	}
	return nil
}

// opts builds a []fieldOption from display names where value == display.
// An entry with display "default" maps to value "".
func opts(names ...string) []fieldOption {
	out := make([]fieldOption, 0, len(names))
	for _, n := range names {
		v := n
		if n == "default" {
			v = ""
		}
		out = append(out, fieldOption{Value: v, Display: n})
	}
	return out
}

// optsInherit builds options with "inherit" as the first choice (value "").
func optsInherit(names ...string) []fieldOption {
	out := make([]fieldOption, 0, len(names)+1)
	out = append(out, fieldOption{Value: "", Display: "inherit"})
	for _, n := range names {
		out = append(out, fieldOption{Value: n, Display: n})
	}
	return out
}
