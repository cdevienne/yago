package main

import (
	"fmt"

	"github.com/aacanakin/qb"

	"bitbucket.org/cdevienne/yagorm"
)

func main() {
	meta := yagorm.NewMetadata()
	meta.AddMapper(&PersonMapper{})

	s := yagorm.Select(meta, &Person{})
	s.GroupBy()

	engine, err := qb.NewEngine("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	engine.SetDialect(qb.NewDialect("sqlite3"))

	meta.GetQbMetadata().CreateAll(engine)

	db := yagorm.New(meta, engine)

	p := NewPerson()
	p.Name = "Toto"
	db.Save(p)

	p = NewPerson()
	p.Name = "Titi"
	db.Save(p)

	q := db.Query(&Person{})
	q = q.Where(db.Metadata.GetMapper(&Person{}).Table().C("name").Eq("Titi"))

	p = &Person{}
	if err := q.One(p); err != nil {
		panic(err)
	}
	fmt.Println(p.Name)
}
