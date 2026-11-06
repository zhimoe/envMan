package app

func (a *App) takeSnapshot(scope Scope, note string) (Snapshot, error) {
	variables, err := a.env.Load(scope)
	if err != nil {
		return Snapshot{}, err
	}
	return a.store.SaveSnapshot(scope, note, variables)
}
