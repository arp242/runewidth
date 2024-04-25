runewidth provides functions to get fixed width of the character or string.

This is a fork of https://github.com/mattn/go-runewidth, updated to the newest
Unicode. It also removes various helper functions, so all that remains is just
the `runewidth.RuneWidth()` function:

    runewidth.RuneWidth('a')
    runewidth.RuneWidth('つ')
    runewidth.RuneWidth('🤷')

Note this can NOT be used to get the width of the string:

    // Broken! Do not do this.
    l := 0
    for _, r := range str {
        l += runewidth.RuneWidth(r)
    }

Use https://github.com/arp242/termtext or https://github.com/rivo/uniseg for
getting the width of a string.

This is mostly useful in conjunction with the uniseg package, as used in
termtext.
