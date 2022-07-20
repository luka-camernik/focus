# focus

*A run-or-raise application switcher for any X11 desktop*

The idea is simple — bind a key for any given application that will:

- focus the application's most recently opened window, if it is running
- with additional flags (-o) launch the application, if it's not already running.
- Using command agin will cycle to the application's next window, if there's more than one and one is focused already (this only works when use as keybind).


## Synopsis

    Usage: focus -p APPLICATION [ARG]...
    Example: `focus -p firefox -o` (this will focus firefox OR open if no window was found)
	
    Options:
      -o  Open new window if none found
      -op Open a specified program when it cannot be found (for example `focus sublime -o -op subl`)
      -p  Which program to attempt to focus (Required)
      -v  Print version number.
