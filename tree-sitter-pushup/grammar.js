/**
 * @file Pushup is a web framework for the Go programming language
 * @author Paul Smith <paul@adhoc.team>
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const html_grammar = require("tree-sitter-html/grammar");
const go_grammar = require("tree-sitter-go/grammar");

module.exports = grammar({
  name: "pushup",

  externals: ($) => [
    $._start_tag_name,
    $._script_start_tag_name,
    $._style_start_tag_name,
    $._end_tag_name,
    $.raw_text,
    $.comment,
  ],

  extras: ($) => [/\s+/],

  //conflicts: ($) => [
  //  [$._node, $.element],
  //  [$.attribute, $._expression],
  //],

  rules: {
    document: ($) => repeat($._node),

    _node: ($) =>
      choice(
        $.element,
        $.text,
        $.comment,
        $.go_code_block,
        $.go_expression,
        $.partial_block,
        $.param_declaration,
        $.import_statement,
        $.if_statement,
        $.for_statement,
      ),

    element: ($) =>
      seq("<", $._start_tag_name, repeat($.attribute), choice(">", "/>")),

    attribute: ($) =>
      seq(
        $.attribute_name,
        optional(seq("=", choice($.attribute_value, $.go_expression))),
      ),

    attribute_name: ($) => /[a-zA-Z_][a-zA-Z0-9_\-:]*/,

    attribute_value: ($) =>
      choice(
        seq('"', optional(/[^"]+/), '"'),
        seq("'", optional(/[^']+/), "'"),
      ),

    text: ($) => /[^<^]+/,

    go_code_block: ($) =>
      seq(
        "^{",
        repeat(
          choice(
            /[^}]/,
            seq("{", repeat(choice(/[^}]/, seq("{", /[^}]*/, "}"))), "}"),
          ),
        ),
        "}",
      ),

    go_expression: ($) =>
      seq(
        // implicit expression
        seq("^", /[a-zA-Z_][a-zA-Z0-9_\.]*/),
        // explicit expression
        seq("^(", repeat(/[^)]/), ")"),
      ),

    partial_block: ($) =>
      seq("^partial", $.identifier, "{", repeat($._node), "}"),

    param_declaration: ($) =>
      choice(
        // declaration form
        seq("^param", $.identifier, $.type_expression),
        // use form
        seq("^param(", $.identifier, ")"),
      ),

    import_statement: ($) =>
      seq("^import", optional(choice(".", $.identifier)), $.string_literal),

    if_statement: ($) =>
      seq(
        "^if",
        $.go_expression,
        "{",
        repeat($._node),
        "}",
        optional(
          seq("^else", choice(seq("{", repeat($._node), "}"), $.if_statement)),
        ),
      ),

    for_statement: ($) =>
      seq("^for", $.go_expression, "{", repeat($._node), "}"),

    identifier: ($) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    type_expression: ($) =>
      seq(optional("*"), $.identifier, optional(seq(".", $.identifier))),

    string_literal: ($) => /"[^"]*"/,
  },
});
