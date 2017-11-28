package test_files

type Struct struct {
	Field  string `tag1:"looooooooongtag1" tag2:"val2"`
	Field2 string `tag1:"shorttag"         tag2:"val34"`
	Field3 string `tag1:"short"                         tag3:"val56"`
	Field4 string `                                                   tag4:"val78"`
}