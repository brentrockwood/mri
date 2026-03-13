import { useAnalysis } from './hooks/useAnalysis'

export function App() {
  const analysis = useAnalysis()
  return (
    <div>
      <h1>{analysis.repo.name}</h1>
      <p>{analysis.repo.module_count} modules</p>
    </div>
  )
}
