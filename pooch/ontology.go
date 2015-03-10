package pooch

func ontologyServerGetIn(tl *Tasklist) []OntologyNodeIn {
	ontology := tl.GetOntology()
	knownTags := map[string]bool{}

	getTagsInOntology(knownTags, ontology)

	savedSearches := tl.GetSavedSearches()
	tags := tl.GetTags()

	for _, ss := range savedSearches {
		n := "#%" + ss
		if _, ok := knownTags[n]; !ok {
			ontology = append(ontology, OntologyNodeIn{n, "open", nil})
		}
	}

	for _, t := range tags {
		n := "#" + t
		if _, ok := knownTags[n]; !ok {
			ontology = append(ontology, OntologyNodeIn{n, "open", nil})
		}
	}

	return ontology
}

func ontologyFindParent(ontology *[]OntologyNodeIn, n string) (*[]OntologyNodeIn, int) {
	for i := range *ontology {
		if (*ontology)[i].Data == n {
			return ontology, i
		}
		r, j := ontologyFindParent(&((*ontology)[i].Children), n)
		if r != nil {
			return r, j
		}
	}
	return nil, -1
}

func ontologyMoveSibling(src, dst string, ontology []OntologyNodeIn) []OntologyNodeIn {
	if src == dst {
		return ontology
	}

	p, i := ontologyFindParent(&ontology, src)
	srcNode := (*p)[i]
	copy((*p)[i:], (*p)[i+1:])
	*p = (*p)[:len(*p)-1]

	p, i = ontologyFindParent(&ontology, dst)
	*p = append(*p, srcNode)
	copy((*p)[i+2:], (*p)[i+1:len(*p)-1])
	(*p)[i+1] = srcNode

	return ontology
}

func ontologyMoveChildren(src, dst string, ontology []OntologyNodeIn) []OntologyNodeIn {
	if src == dst {
		return ontology
	}

	p, i := ontologyFindParent(&ontology, src)
	srcNode := (*p)[i]
	copy((*p)[i:], (*p)[i+1:])
	*p = (*p)[:len(*p)-1]

	p, i = ontologyFindParent(&ontology, dst)
	(*p)[i].Children = append((*p)[i].Children, srcNode)

	return ontology
}
