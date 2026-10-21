package wasmbinding

// OverrideWhitelistedQuery swaps the response factory registered for path and
// returns a function that puts the previous entry back.
func OverrideWhitelistedQuery(path string, factory ProtoMessageFactory) (restore func()) {
	previous, existed := stargateWhitelist.Load(path)
	stargateWhitelist.Store(path, factory)
	return func() {
		if existed {
			stargateWhitelist.Store(path, previous)
			return
		}
		stargateWhitelist.Delete(path)
	}
}
