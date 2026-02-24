import * as React from "react"

export type PanelType = "logs" | "terminal" | "files"
export type TargetType = "pod" | "node"

export interface PanelState {
  type: PanelType
  targetType?: TargetType
  appId: string
  appName: string
  instanceName: string
  containerName: string
  containers: string[]
  initContainers?: string[]
}

interface BottomPanelContextValue {
  isOpen: boolean
  isMinimized: boolean
  panelState: PanelState | null
  
  openPanel: (state: PanelState) => void
  closePanel: () => void
  minimizePanel: () => void
  maximizePanel: () => void
  toggleMinimize: () => void
  switchType: (type: PanelType) => void
  switchContainer: (containerName: string) => void
  switchInstance: (instanceName: string, containers: string[], initContainers?: string[]) => void
}

const BottomPanelContext = React.createContext<BottomPanelContextValue | null>(null)

export function BottomPanelProvider({ children }: { children: React.ReactNode }) {
  const [isOpen, setIsOpen] = React.useState(false)
  const [isMinimized, setIsMinimized] = React.useState(false)
  const [panelState, setPanelState] = React.useState<PanelState | null>(null)

  const openPanel = React.useCallback((state: PanelState) => {
    setPanelState(state)
    setIsOpen(true)
    setIsMinimized(false)
  }, [])

  const closePanel = React.useCallback(() => {
    setIsOpen(false)
    setIsMinimized(false)
    setPanelState(null)
  }, [])

  const minimizePanel = React.useCallback(() => {
    setIsMinimized(true)
  }, [])

  const maximizePanel = React.useCallback(() => {
    setIsMinimized(false)
  }, [])

  const toggleMinimize = React.useCallback(() => {
    setIsMinimized(prev => !prev)
  }, [])

  const switchType = React.useCallback((type: PanelType) => {
    if (panelState) {
      setPanelState({ ...panelState, type })
    }
  }, [panelState])

  const switchContainer = React.useCallback((containerName: string) => {
    if (panelState) {
      setPanelState({ ...panelState, containerName })
    }
  }, [panelState])

  const switchInstance = React.useCallback((instanceName: string, containers: string[], initContainers?: string[]) => {
    if (panelState) {
      const allNewContainers = [...(initContainers || []), ...containers]
      const newContainerName = allNewContainers.includes(panelState.containerName)
        ? panelState.containerName
        : (containers[0] || panelState.containerName)

      setPanelState({
        ...panelState,
        instanceName,
        containers,
        initContainers,
        containerName: newContainerName
      })
    }
  }, [panelState])

  const value = React.useMemo(() => ({
    isOpen,
    isMinimized,
    panelState,
    openPanel,
    closePanel,
    minimizePanel,
    maximizePanel,
    toggleMinimize,
    switchType,
    switchContainer,
    switchInstance,
  }), [
    isOpen,
    isMinimized,
    panelState,
    openPanel,
    closePanel,
    minimizePanel,
    maximizePanel,
    toggleMinimize,
    switchType,
    switchContainer,
    switchInstance,
  ])

  return (
    <BottomPanelContext.Provider value={value}>
      {children}
    </BottomPanelContext.Provider>
  )
}

export function useBottomPanel() {
  const context = React.useContext(BottomPanelContext)
  if (!context) {
    throw new Error("useBottomPanel must be used within a BottomPanelProvider")
  }
  return context
}
