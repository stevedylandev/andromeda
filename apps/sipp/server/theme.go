package server

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
)

// Darkmatter chroma style — base16 template filled with the "Darkmatter" scheme.
//
//	base00 #121113  base01 #121212  base02 #222222  base03 #333333
//	base04 #999999  base05 #c1c1c1  base06 #999999  base07 #c1c1c1
//	base08 #5f8787  base09 #aaaaaa  base0A #e78a53  base0B #fbcb97
//	base0C #aaaaaa  base0D #888888  base0E #999999  base0F #444444
func init() {
	styles.Register(chroma.MustNewStyle("darkmatter", chroma.StyleEntries{
		chroma.Other:                    "#c1c1c1",
		chroma.Error:                    "#5f8787",
		chroma.Background:               "bg:#121113",
		chroma.Keyword:                  "#999999",
		chroma.KeywordConstant:          "#999999",
		chroma.KeywordDeclaration:       "#5f8787",
		chroma.KeywordNamespace:         "#999999",
		chroma.KeywordPseudo:            "#999999",
		chroma.KeywordReserved:          "#999999",
		chroma.KeywordType:              "#aaaaaa",
		chroma.Name:                     "#c1c1c1",
		chroma.NameAttribute:            "#888888",
		chroma.NameBuiltin:              "#5f8787",
		chroma.NameBuiltinPseudo:        "#c1c1c1",
		chroma.NameClass:                "#e78a53",
		chroma.NameConstant:             "#aaaaaa",
		chroma.NameDecorator:            "#aaaaaa",
		chroma.NameEntity:               "#c1c1c1",
		chroma.NameException:            "#c1c1c1",
		chroma.NameFunction:             "#888888",
		chroma.NameLabel:                "#5f8787",
		chroma.NameNamespace:            "#c1c1c1",
		chroma.NameOther:                "#c1c1c1",
		chroma.NameTag:                  "#999999",
		chroma.NameVariable:             "#5f8787",
		chroma.NameVariableClass:        "#5f8787",
		chroma.NameVariableGlobal:       "#5f8787",
		chroma.NameVariableInstance:     "#5f8787",
		chroma.Literal:                  "#c1c1c1",
		chroma.LiteralDate:              "#c1c1c1",
		chroma.LiteralString:            "#fbcb97",
		chroma.LiteralStringBacktick:    "#fbcb97",
		chroma.LiteralStringChar:        "#fbcb97",
		chroma.LiteralStringDoc:         "#fbcb97",
		chroma.LiteralStringDouble:      "#fbcb97",
		chroma.LiteralStringEscape:      "#fbcb97",
		chroma.LiteralStringHeredoc:     "#fbcb97",
		chroma.LiteralStringInterpol:    "#fbcb97",
		chroma.LiteralStringOther:       "#fbcb97",
		chroma.LiteralStringRegex:       "#fbcb97",
		chroma.LiteralStringSingle:      "#fbcb97",
		chroma.LiteralStringSymbol:      "#fbcb97",
		chroma.LiteralNumber:            "#aaaaaa",
		chroma.LiteralNumberBin:         "#aaaaaa",
		chroma.LiteralNumberFloat:       "#aaaaaa",
		chroma.LiteralNumberHex:         "#aaaaaa",
		chroma.LiteralNumberInteger:     "#aaaaaa",
		chroma.LiteralNumberIntegerLong: "#aaaaaa",
		chroma.LiteralNumberOct:         "#aaaaaa",
		chroma.Operator:                 "#999999",
		chroma.OperatorWord:             "#999999",
		chroma.Punctuation:              "#c1c1c1",
		chroma.Comment:                  "#333333",
		chroma.CommentHashbang:          "#333333",
		chroma.CommentMultiline:         "#333333",
		chroma.CommentSingle:            "#333333",
		chroma.CommentSpecial:           "#333333",
		chroma.CommentPreproc:           "#333333",
		chroma.Generic:                  "#c1c1c1",
		chroma.GenericDeleted:           "#5f8787",
		chroma.GenericEmph:              "underline #c1c1c1",
		chroma.GenericError:             "#5f8787",
		chroma.GenericHeading:           "bold #c1c1c1",
		chroma.GenericInserted:          "bold #c1c1c1",
		chroma.GenericOutput:            "#222222",
		chroma.GenericPrompt:            "#c1c1c1",
		chroma.GenericStrong:            "italic #c1c1c1",
		chroma.GenericSubheading:        "bold #c1c1c1",
		chroma.GenericTraceback:         "#c1c1c1",
		chroma.GenericUnderline:         "underline",
		chroma.Text:                     "#c1c1c1",
		chroma.TextWhitespace:           "#c1c1c1",
	}))
}
