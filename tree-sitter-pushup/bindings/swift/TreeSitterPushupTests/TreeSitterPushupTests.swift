import XCTest
import SwiftTreeSitter
import TreeSitterPushup

final class TreeSitterPushupTests: XCTestCase {
    func testCanLoadGrammar() throws {
        let parser = Parser()
        let language = Language(language: tree_sitter_pushup())
        XCTAssertNoThrow(try parser.setLanguage(language),
                         "Error loading Pushup grammar")
    }
}
