Do not do this work using claude 4 opus, instead wait until we run out of tokens and assign this work to a lower quality model.


i do not like how we use snake case on variable names. we should be using camelCase as that is idiomatic for go. The only times we should be using snake case is to differentiate between generated golang code, or a C file / function. let's review all file/function/variable names and ensure we are using camelCase.

Let's build linting into our makefile for the golang part of our codebase.
