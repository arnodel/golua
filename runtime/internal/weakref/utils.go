package weakref

// Convenience method run by the Pool implementations
func runPrefinalizers(ifaces []interface{}) []interface{} {
	for _, iface := range ifaces {
		if pf, ok := iface.(Prefinalizer); ok {
			pf.Prefinalize()
		}
	}
	return ifaces
}
