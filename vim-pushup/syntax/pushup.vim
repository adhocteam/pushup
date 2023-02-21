" TODO: Implicit expressions are barely highlighted. It would be better if
" they were. Philosophically this syntax file defers to the go and html syntax
" files by marking those regions as transparent. However the go syntax barely
" highlights expressions by default. For now they are highlighted as a single
" color, which seems to be the better short-term trade-off. Two options for a
" longer-term fix:
"   * Take advantage of the limited syntax of implicit expressions and match
"     the contents with specific rules in this file. Potentially add an option
"     for whether to highlight the full expression as a unit versus
"     highlighting the sub-parts.
"   * Temporarily override some of the g:go_highlight_* config variables when
"     including the go syntax to ensure that sub-parts of the implicit
"     expressions are highlighted in pushup files even if they are not
"     highlighted in the user's normal go code.

" TODO: Many of the pushup expressions that are expected to be embedded into
" html are defined as top-level elements and also added to the htmlTop
" cluster. That is an intentional extension-point provided by the standard
" html syntax file. Since the <head> and <title> elements are constrained in
" what can be included, the @htmlTop cluster is not contained within them. In
" order to make pushup expressions highlight within those this uses
" containedin=htmlHead,htmlTitle. This would be better fixed by a patch
" upstream to introduce an additional extension cluster that could be matched
" within all elements that have a contains= option.

if exists('b:current_syntax')
    unlet b:current_syntax
endif
runtime syntax/html.vim

if exists('b:current_syntax')
    unlet b:current_syntax
endif
syntax include @golang syntax/go.vim

syn keyword pushupForKey for contained
syn keyword pushupHandlerKey handler contained
syn keyword pushupIfKey if contained
syn keyword pushupPartialKey partial contained nextgroup=pushupPartialName skipwhite
syn keyword pushupSectionKey section contained nextgroup=pushupSectionName skipwhite

syn region pushupBlock start=/\^{/me=e-1 end=/}/ skip=/\([^}]\|\\[}]\)/ contains=pushupTranSym,pushupGoBlock transparent
syn region pushupImplExpr start=/\^\(for\|if\|section\|handler\|[{(]\)\@!/ skip=/[.\w()]/ end=/\W/me=e-1 contains=pushupTranSym,@golang
syn region pushupExplExpr start=/\^(/rs=s+1 end=/)/ skip=/\([^)]\|\\[)]\)/ extend contains=pushupGoParenExpr contained

" Since expression syntax is more generic than directive syntax and both are
" regions, this needs to be defined after the expression rules.
syn keyword pushupDirName import contained
syn region pushupDirSimpl start=/\^\(import\)/ end=/$/ extend skipwhite matchgroup=NONE contains=pushupTranSym,pushupDirName,@golang nextgroup=pushupTranSym

" htmlTop is defined by the standard vim HTML syntax file. This extends the
" cluster of top-level identifiers, which allows them to be matched inside the
" HTML tags.
syn cluster htmlTop add=pushupExplExpr,pushupImplExpr,pushupIf,pushupFor,pushupPartial

syn region pushupHandler start=/\^handler\s\+{/ end=/}/ skipwhite skipnl contains=pushupTranSym,pushupHandlerKey,pushupGoBlock nextgroup=pushupTranSym transparent

" pushupIfPrefix is required to ensure that the pushupIfCond only matches in
" the intended position and does not match within the braces.
syn match pushupIfPrefix /\^if\s\+.\{-}{/he=e-1 contained contains=pushupTranSym,pushupIfKey,pushupIfCond nextgroup=pushupIfKey
" pushupElse cannot be a keyword because it would match within HTML blocks
" that exist within if blocks.
syn match pushupElse /}\s\+\zselse\ze\s\+{/ contained
syn region pushupIf start=/\^if\s\+.\{-}{/ end=/}\(\s\+else\s\+{\)?/ skipwhite skipnl contains=pushupIfPrefix,pushupElse,@htmlTop nextgroup=pushupIfPrefix transparent containedin=htmlTitle,htmlHead
syn match pushupIfCond /[^{]\+/ contained contains=@golang

syn match pushupForExpr /[^{]\+/ contained contains=@golang
syn match pushupForPrefix /\^for\s\+.\{-}{/ contained contains=pushupTranSym,pushupForKey,pushupForExpr transparent
syn region pushupFor start=/\^for\s\+.\{-}{/ end=/}/ skipwhite contains=pushupForPrefix,@htmlTop nextgroup=pushupTranSym transparent containedin=htmlTitle,htmlHead

" pushupGoParenExpr and pushupGoBlock exist only to wrap existing go syntax
" rules with the 'extend' flag so that the pushup regions can be ambiguously
" defined with reliable practical results.
" syn match pushupParenOp /[()]/ contained
syn region pushupGoParenExpr start=/(/ms=s+1 end=/)/me=e-1 extend contained contains=@golang

" The ms and me adjustments are meant to ensure the goBlock region doesn't
" match these braces.
syn region pushupGoBlock start="{"ms=s+1 end="}"me=e-1 extend contained contains=@golang transparent

syn match pushupPartialName /\s\zs\w\+\ze\s/ contained extend
syn match pushupPartialPrefix /\^partial\s\+\S\+\s*{/ extend contained contains=pushupTranSym,pushupPartialKey,pushupPartialName nextgroup=pushupTranSym
syn region pushupPartial start=/\^partial\s\+\S\+\s*{/ end=/}/ skip=/[^}]/ contains=@htmlTop,pushupPartialPrefix nextgroup=pushupPartialPrefix skipwhite skipnl transparent

syn match pushupSectionName /\s\zs\w\+\ze\s/ contained extend
syn match pushupSectionPrefix /\^section\s\+\S\+\s*{/ extend contained contains=pushupTranSym,pushupSectionKey,pushupSectionName nextgroup=pushupTranSym
syn region pushupSection start=/\^section\s\+\S\+\s*{/ end=/}/ skip=/[^}]/ contains=@htmlTop,pushupSectionPrefix nextgroup=pushupSectionPrefix skipwhite skipnl transparent

" This syn-keyword approach seems like it should work. The caret character
" does appear in echo &iskeyword. However the goOperator syn-match rule seems
" to take precedence. That seems plainly at odds with syn-priority though, so
" I probably just did it wrong.
" set iskeyword+=^
" syn keyword pushupTranSym ^
syn match pushupTranSym /\^/ contained

syn sync fromstart

highlight pushupCatchall guifg=yellow
highlight link pushupDirSimpl pushupCatchall
highlight link pushupPartialName Identifier
highlight link pushupSectionName Identifier
highlight link pushupImplExpr Underlined
highlight link pushupExplExpr Underlined

" highlight pushupTranSym guifg=green
highlight link pushupTranSym Operator

highlight pushupBlockDelims guifg=magenta

" highlight pushupKeyword guifg=lightgreen
highlight link pushupKeyword Keyword
highlight link pushupForKey pushupKeyword
highlight link pushupIfKey pushupKeyword
highlight link pushupElse pushupKeyword
highlight link pushupHandlerKey pushupKeyword
highlight link pushupPartialKey pushupKeyword
highlight link pushupSectionKey pushupKeyword
highlight link pushupDirName pushupKeyword

let b:current_syntax = 'pushup'
