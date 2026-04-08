//#region src/lib/compare.ts
function compareQuery(a, b) {
	if (a === b) return true;
	if (a === null || b === null) return false;
	if (typeof a === "string" || typeof b === "string") return false;
	if (a.length !== b.length) return false;
	return a.every((value, index) => value === b[index]);
}

//#endregion
//#region src/testing.ts
/**
* Test that a parser is bijective (serialize then parse gives back the same value).
*
* It will throw if the parser does not serialize the input to the expected serialized value,
* or if the parser does not parse the serialized value to the expected input value.
* The parser's `eq` function (if provided, otherwise `===`) is used to compare the values.
*
* Usage:
* ```ts
* // Expect it to pass (no error thrown)
* expect(isParserBijective(parseAsInteger, '42', 42)).toBe(true)
* // Expect it to fail
* expect(() => isParserBijective(parseAsInteger, '42', 47)).toThrow()
* ```
*
* @param parser The parser to test
* @param serialized The serialized representation of the input to test against
* @param input An input value to test against
* @returns `true` if the test passes, otherwise it will throw.
*/
function isParserBijective(parser, serialized, input) {
	if (parser.type === "multi" && Array.isArray(serialized)) {
		testSerializeThenParse(parser, input);
		testParseThenSerialize(parser, serialized);
	} else if (parser.type !== "multi" && typeof serialized === "string") {
		testSerializeThenParse(parser, input);
		testParseThenSerialize(parser, serialized);
	} else throw new Error(`[nuqs] isParserBijective: mismatched parser type and serialized value type`);
	if (!compareQuery(parser.serialize(input), serialized)) throw new Error(`[nuqs] parser.serialize does not match expected serialized value
  Expected: '${serialized}'
  Received: '${parser.serialize(input)}'
  `);
	const parsed = parser.parse(serialized);
	if (!parser.eq(parsed, input)) throw new Error(`[nuqs] parser.parse does not match expected input value
  Expected: ${input}
  Received: ${parsed}
  `);
	return true;
}
/**
* Test that a parser is bijective (serialize then parse gives back the same value).
*
* It will throw if the parser is not bijective (if the parsed value is not equal to the input value).
* The parser's `eq` function is used to compare the values.
*
* Usage:
* ```ts
* // Expect it to pass (no error thrown)
* expect(testSerializeThenParse(myParser, 'foo')).toBe(true)
* // Expect it to fail
* expect(() => testSerializeThenParse(myParser, 'bar')).toThrow()
* ```
*
* @param parser The parser to test
* @param input An input value to test against
* @returns `true` if the test passes, otherwise it will throw.
*/
function testSerializeThenParse(parser, input) {
	const serialized = parser.serialize(input);
	const parsed = parser.type == "multi" && Array.isArray(serialized) ? parser.parse(serialized) : parser.type !== "multi" && typeof serialized === "string" ? parser.parse(serialized) : null;
	if (parsed === null) throw new Error(`[nuqs] testSerializeThenParse: parsed value is null (when parsing ${serialized} serialized from ${input})`);
	if (!parser.eq(input, parsed)) throw new Error(`[nuqs] parser is not bijective (in testSerializeThenParse)
  Expected value:         ${typeof input === "object" ? JSON.stringify(input) : input}
  Received parsed value:  ${typeof parsed === "object" ? JSON.stringify(parsed) : parsed}
  Serialized as: '${serialized}'
  `);
	return true;
}
/**
* Tests that a parser is bijective (parse then serialize gives back the same query string).
*
* It will throw if the parser is not bijective (if the serialized value is not equal to the input query).
*
* Usage:
* ```ts
* // Expect it to pass (no error thrown)
* expect(testParseThenSerialize(myParser, 'foo')).toBe(true)
* // Expect it to fail
* expect(() => testParseThenSerialize(myParser, 'bar')).toThrow()
* ```
*
* @param parser The parser to test
* @param input A query string to test against
* @returns `true` if the test passes, otherwise it will throw.
*/
function testParseThenSerialize(parser, input) {
	const parsed = parser.type === "multi" && Array.isArray(input) ? parser.parse(input) : parser.type !== "multi" && typeof input === "string" ? parser.parse(input) : null;
	if (parsed === null) throw new Error(`[nuqs] testParseThenSerialize: parsed value is null (when parsing ${input})`);
	const serialized = parser.serialize(parsed);
	if (!compareQuery(serialized, input)) throw new Error(`[nuqs] parser is not bijective (in testParseThenSerialize)
  Expected query: '${input}'
  Received query: '${serialized}'
  Parsed value: ${parsed}
`);
	return true;
}

//#endregion
export { isParserBijective, testParseThenSerialize, testSerializeThenParse };
//# sourceMappingURL=testing.js.map