package generate

import (
	"text/template"
)

// FileData contains top-level infos for templates
type FileData struct {
	Package string
	Imports map[string]bool
}

// ColumnTags contains tags set on the fields
type ColumnTags struct {
	ColumnName    string
	PrimaryKey    bool
	AutoIncrement bool
	ForeignKeys   []string
	Indexes       []string
	UniqueIndexes []string
}

// FieldData describes a field to be mapped
type FieldData struct {
	Tags            ColumnTags
	Name            string
	Type            string
	EmptyValue      string
	ColumnName      string
	ColumnType      string
	ColumnModifiers string
	ColumnNameConst string
}

type FKData struct {
	Column    *FieldData
	RefTable  *StructData
	RefColumn *FieldData
	OnUpdate  string
	OnDelete  string
}

// StructData describes a struct to be mapped
type StructData struct {
	Name              string
	PrivateBasename   string
	TableName         string
	Fields            []FieldData
	PKeyFields        []*FieldData
	AutoIncrementPKey *FieldData

	Indexes       map[string][]int
	UniqueIndexes map[string][]int
	ForeignKeys   []FKData

	File FileData
}

var (
	prologTemplate = template.Must(template.New("prolog").Parse(
		`package {{ .Package }}

// generated with yago. Better NOT to edit!

import (
	"database/sql"
	"reflect"

	"github.com/aacanakin/qb"

	"github.com/orus-io/yago"

	{{- range $k, $_ := .Imports }}
	"{{$k}}"
	{{- end }}
)
`))

	structTemplate = template.Must(template.New("struct").Parse(
		`{{ $root := . }}{{ $Struct := .Name }}{{ $Table := printf "%s%s" .PrivateBasename "Table" }}
const (
	// {{ .Name }}TableName is the {{ .Name }} associated table name
	{{ .Name }}TableName = "{{ .TableName }}"
{{- range .Fields }}
	// {{ .ColumnNameConst }} is the {{ .Name }} field associated column name
	{{ .ColumnNameConst }} = "{{ .ColumnName }}"
{{- end }}
)

var {{ $Table }} = qb.Table(
	{{ .Name }}TableName,
	{{- range .Fields }}
	qb.Column({{ .ColumnNameConst }}, {{ .ColumnType }}){{ .ColumnModifiers }},
	{{- end }}
	{{- range .UniqueIndexes }}
	qb.UniqueKey(
		{{- range . }}
		{{ (index $root.Fields .).ColumnNameConst }},
		{{- end }}
	),{{- end }} {{- range .ForeignKeys }}
	qb.ForeignKey({{ .Column.ColumnNameConst }}).References({{ .RefTable.Name }}TableName, {{ .RefColumn.ColumnNameConst }}){{- if .OnUpdate }}.OnUpdate("{{ .OnUpdate }}"){{- end}}{{- if .OnDelete }}.OnDelete("{{ .OnDelete }}"){{- end }},{{- end }}
){{- range $name, $cols := .Indexes }}.Index(
	{{- range . }}
	{{ (index $root.Fields .).ColumnNameConst }},
	{{- end }}
){{- end }}

var {{ .PrivateBasename }}Type = reflect.TypeOf({{ .Name }}{})

// StructType returns the reflect.Type of the struct
// It is used for indexing mappers (and only that I guess, so
// it could be replaced with a unique identifier).
func ({{ .Name }}) StructType() reflect.Type {
	return {{ .PrivateBasename }}Type
}

// {{ .Name }}Model
type {{ .Name }}Model struct {
	mapper *{{ .Name }}Mapper
	{{- range .Fields }}
	{{ .Name }} yago.ScalarField
	{{- end }}
}

func New{{ .Name }}Model(meta *yago.Metadata) {{ .Name }}Model {
	mapper := New{{ .Name }}Mapper()
	meta.AddMapper(mapper)
	return {{ .Name }}Model {
		mapper: mapper,
		{{- range .Fields }}
		{{ .Name }}: yago.NewScalarField(mapper.Table().C({{ .ColumnNameConst }})),
		{{- end }}
	}
}

func (m {{ .Name }}Model) GetMapper() yago.Mapper {
	return m.mapper
}

// New{{ .Name }}Mapper initialize a New{{ .Name }}Mapper
func New{{ .Name }}Mapper() *{{ .Name }}Mapper {
	m := &{{ .Name }}Mapper{}
	return m
}

// {{ .Name }}Mapper is the {{ .Name }} mapper
type {{ .Name }}Mapper struct{}

// Name returns the mapper name
func (*{{ .Name }}Mapper) Name() string {
	return "{{ .File.Package }}/{{ .Name }}"
}

// Table returns the mapper table
func (*{{ .Name }}Mapper) Table() *qb.TableElem {
	return &{{ $Table }}
}

// StructType returns the reflect.Type of the mapped structure
func ({{ .Name }}Mapper) StructType() reflect.Type {
	return {{ .PrivateBasename }}Type
}

// SQLValues returns values as a map
// The primary key is included only if having non-default values
func (mapper {{ .Name }}Mapper) SQLValues(instance yago.MappedStruct) map[string]interface{} {
	s, ok := instance.(*{{ .Name }})
	if !ok {
		panic("Wrong struct type passed to the mapper")
	}
	m := make(map[string]interface{})
	{{- range .PKeyFields }}
	if s.{{ .Name }} != {{ .EmptyValue }} {
		m[{{ .ColumnNameConst }}] = s.{{ .Name }}
	}
	{{- end }}
	{{- range .Fields }}
	{{- if not .Tags.PrimaryKey }}
	m[{{ .ColumnNameConst }}] = s.{{ .Name }}
	{{- end }}
	{{- end }}
	return m
}

// FieldList returns the list of fields for a select
func (mapper {{ .Name }}Mapper) FieldList() []qb.Clause {
	return []qb.Clause{
		{{- range .Fields }}
		{{ $Table }}.C({{ .ColumnNameConst }}),
		{{- end }}
	}
}

// Scan a struct
func (mapper {{ .Name }}Mapper) Scan(rows *sql.Rows, instance yago.MappedStruct) error {
	s, ok := instance.(*{{ .Name }})
	if !ok {
		panic("Wrong struct type passed to the mapper")
	}
	return rows.Scan(
	{{- range $_, $fd := .Fields }}
		&s.{{ $fd.Name }},
	{{- end }}
	)
}

// AutoIncrementPKey return true if a column of the pkey is autoincremented
func ({{ .Name }}Mapper) AutoIncrementPKey() bool {
	{{- if .AutoIncrementPKey }}
	return true
	{{- else }}
	return false
	{{- end }}
}

// LoadAutoIncrementPKeyValue set the pkey autoincremented column value
func ({{ .Name }}Mapper) LoadAutoIncrementPKeyValue(instance yago.MappedStruct, value int64) {
	{{- if .AutoIncrementPKey }}
	s := instance.(*{{ $Struct }})
	s.{{ .AutoIncrementPKey.Name }} = value
	{{- else }}
	panic("{{ .Name }} has no auto increment column in its pkey")
	{{- end }}
}

// PKeyClause returns a clause that matches the instance primary key
func (mapper {{ .Name }}Mapper) PKeyClause(instance yago.MappedStruct) qb.Clause {
	{{- if eq 1 (len .PKeyFields) }}
	return {{ $Table }}.C({{ (index .PKeyFields 0).ColumnNameConst }}).Eq(instance.(*{{ .Name }}).{{ (index .PKeyFields 0).Name }})
	{{- else }}
	return qb.And(
		{{- range .PKeyFields }}
		{{ $Table }}.C({{ .ColumnNameConst }}).Eq(instance.(*{{ $Struct }}).{{ .Name }}),
		{{- end }}
	)
	{{- end }}
}
`))
)
