if exists("b:current_syntax")
    finish
endif

" runtime! syntax/html.vim
" unlet b:current_syntax

let b:current_syntax = "pushup"

syn include @Go syntax/go.vim

" include '^' in list of characters that can be a keyword
set iskeyword+=^

syn region pushupString start=+"+ end=+"+ contained
syn match pushupBang '!' contained
syn match pushupIdent '\h\w*' contained
syn match pushupBang '!' contained

syn keyword pushupKeyword ^if ^for ^handler
syn keyword pushupKeyword ^import skipwhite nextgroup=pushupString
syn keyword pushupKeyword ^layout skipwhite nextgroup=pushupIdent,pushupBang

hi def link pushupKeyword Statement
hi def link pushupString String
hi def link pushupIdent Identifier
hi def link pushupBang Identifier
