export type TreeItem<T> = T & {
  children?: TreeItem<T>[]
}

/**
 * Groups items into a tree structure based on parentId
 * and then flattens them into a list for rendering.
 *
 * @param items List of items with id and parent_id
 * @param expandedIds Set of IDs that are expanded
 * @returns Flattened list of items respecting expansion state
 */
export function flattenTree<T extends { id: string; parent_task_id?: string; parent_requirement_id?: string; depth: number; created_at: string }>(
  items: T[],
  expandedIds: Set<string>
): T[] {
  if (!items || items.length === 0) return []

  const itemMap = new Map<string, TreeItem<T>>()
  
  // Clone items to avoid mutating original data
  items.forEach(item => {
    itemMap.set(item.id, { ...item, children: [] })
  })

  const roots: TreeItem<T>[] = []
  
  // Single pass to build tree
  items.forEach(item => {
    const node = itemMap.get(item.id)!
    const parentId = item.parent_task_id || item.parent_requirement_id

    // Only attach to parent if parent is present in the current dataset
    if (parentId && itemMap.has(parentId)) {
      const parent = itemMap.get(parentId)!
      parent.children!.push(node)
    } else {
      // It's a root, or its parent is missing from this page/dataset
      roots.push(node)
    }
  })

  // Helper to sort nodes
  // Requirements/Tasks usually sorted by priority/rank or created_at
  // For now, we respect the incoming order as primary, but ensure children are grouped.
  // Actually, let's just traverse in the order they appear in 'roots'.
  
  const result: T[] = []

  const traverse = (nodes: TreeItem<T>[]) => {
    nodes.forEach(node => {
      result.push(node)
      if (expandedIds.has(node.id) && node.children && node.children.length > 0) {
        traverse(node.children)
      }
    })
  }

  traverse(roots)

  return result
}
