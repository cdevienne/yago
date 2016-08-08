package yagorm

import (
	"reflect"

	"github.com/aacanakin/qb"
)

// Mapper links a mapped struct and table definition
type Mapper interface {
	Name() string
	Table() *qb.TableElem
	StructType() reflect.Type
}

// MappedStruct is implemented by all mapped structures
type MappedStruct interface {
	StructType() reflect.Type
}