if exists("b:current_syntax")
    finish
endif

"runtime! syntax/html.vim
"unlet b:current_syntax

"syn include @Go syntax/go.vim
"syn region pushupGoCode start="<<" end=">>" contains=@Go
"syn keyword pushupKeywordGoCode @handler skipwhite nextgroup=pushupGoCode

" include '@' itself in list of characters that can be a keyword
set iskeyword+=@-@

syn region pushupBlock start="{" end="}" contained contains=pushupKeyword

syn keyword pushupKeywordBlock @if @for skipwhite skipnl skipempty nextgroup=pushupBlock

syn region pushupString start=+"+ end=+"+ contained

syn keyword pushupKeywordSimple @import skipwhite nextgroup=pushupString

syn match pushupBang '!' contained
syn match pushupIdent '\h\w*' contained

syn keyword pushupKeywordGoCode @handler

syn keyword pushupKeywordSimple @layout skipwhite nextgroup=pushupBang,pushupIdent

syn match pushupBraces '[{}]'

let b:current_syntax = "pushup"

hi def link pushupKeywordGoCode Statement
hi def link pushupKeywordBlock Statement
hi def link pushupKeywordSimple Statement
hi def link pushupString String
hi def link pushupIdent Identifier
hi def link pushupBang Special
hi pushupBraces ctermfg=red
