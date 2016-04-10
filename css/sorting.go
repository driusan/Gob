package css

// SortStyles will sort the rules on this element according to the CSS spec, which state:s

// 1. Find all declarations that apply too element/property (already done when this is called)
// 2. Sort according to importance (normal or important) and origin (author, user, or user agent). In ascending order of precedence:
//	1. user agent declarations (defaults)
//	2. user normal declrations (don't exist)
//	3. author normal declarations
//	4. author important declarations
//	5. user important declarations (don't exist)
// 3. Sort rules with the same importance and origin by specificity of selector: more specific selectors will override more general ones. Pseudo-elements and pseudo-classes are counted as normal elements and classes, respectively.
// 4. Finally, sort by order specified: if two declarations have the same weight, origin, and specificity, the latter specified wins. Declarations in imported stylesheets are considered to be before any declaration in the style sheet itself
// BUG(driusan): Specificity is not implemented, nor is the final tie break

type byCSSPrecedence []StyleRule

/*
type StyleRule struct {
	Selector CSSSelector
	Name     StyleAttribute
	Value    StyleValue
	//Values   map[StyleAttribute]StyleValue
	src StyleSource
}
*/
func specificityLess(i, j StyleRule) bool {
	if i.src == InlineStyleSrc {
		return true
	}

	iNumIDs := i.Selector.NumberIDs()
	jNumIDs := i.Selector.NumberIDs()
	if iNumIDs != jNumIDs {
		return iNumIDs > jNumIDs
	}

	iNumAttribs := i.Selector.NumberAttrs()
	jNumAttribs := j.Selector.NumberAttrs()
	iNumClasses := i.Selector.NumberClasses()
	jNumClasses := j.Selector.NumberClasses()

	if (iNumAttribs + iNumClasses) != (jNumAttribs + jNumClasses) {
		return (iNumAttribs + iNumClasses) > (jNumAttribs + jNumClasses)
	}
	iNumElements := i.Selector.NumberElements()
	jNumElements := j.Selector.NumberElements()
	iNumPseudo := i.Selector.NumberPseudo()
	jNumPseudo := j.Selector.NumberPseudo()
	if (iNumElements + iNumPseudo) != (jNumElements + jNumPseudo) {
		return (iNumElements + iNumPseudo) > (jNumElements + jNumPseudo)
	}
	return false
}
func (r byCSSPrecedence) Len() int      { return len(r) }
func (r byCSSPrecedence) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r byCSSPrecedence) Less(i, j int) bool {
	if r[i].src == UserAgentSrc {
		// This is a UserAgent stylesheet.
		// Reminder:
		//	1. user agent declarations
		//	2. user normal declrations
		//	3. author normal declarations
		//	4. author important declarations
		//	5. user important declarations

		// User agent stylesheets are less important than any
		// other stylesheet type, so it's never "less"
		if r[j].src != UserAgentSrc {
			return false
		}
		// Order by importance if they're both user agent style
		// sheets
		if r[i].Value.Important == true && r[j].Value.Important == false {
			return true
		} else if r[i].Value.Important == false && r[j].Value.Important == true {
			return false
		}
		// they're both the same importance, so order by specificity
		return specificityLess(r[i], r[j])
	} else if r[i].src == UserSrc {
		// This is a User stylesheet.
		// Reminder:
		//	1. user agent declarations
		//	2. user normal declrations
		//	3. author normal declarations
		//	4. author important declarations
		//	5. user important declarations

		// Always more important than user agent stylesheets
		if r[j].src == UserAgentSrc {
			return true
		}
		// important user stylesheets are more important
		// than anything else, so always "less"
		if r[i].Value.Important == true {
			// they're both important user sheets, so use
			// specificity as a tie breaker
			if r[j].src == UserSrc && r[j].Value.Important == true {
				return specificityLess(r[i], r[j])
			}
			return true
		}

		// all that's left is author stylesheets, and user normal
		// stylesheets are never more important, so they're never
		// "less"
		return false
	} else if r[i].src == AuthorSrc {
		// This is an Author stylesheet.
		// Reminder:
		//	1. user agent declarations
		//	2. user normal declrations
		//	3. author normal declarations
		//	4. author important declarations
		//	5. user important declarations

		// Always more important than UserAgent stylesheets
		if r[j].src == UserAgentSrc {
			return true
		}

		// User important are more important, but user !important
		// are less important than author stylesheets
		if r[j].src == UserSrc {
			if r[j].Value.Important == true {
				return false
			} else {
				return true
			}
		}

		// everything other than author specified has already
		// been sorted. All that's left is important or not
		// for author stylesheets
		if r[i].Value.Important == true && r[j].Value.Important == false {
			return true
		} else if r[i].Value.Important == false && r[j].Value.Important == true {
			return false
		}

		// both same importance author stylesheets. specificity is the
		// tie breaker
		return specificityLess(r[i], r[j])
	}
	panic("Unhandled stylesheet precedence")
}
