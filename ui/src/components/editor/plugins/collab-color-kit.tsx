'use client'

import { FontBackgroundColorPlugin, FontColorPlugin } from "@platejs/basic-styles/react"

import { CollabColorLeaf } from "@/components/editor/collab-editor-elements"

export const CollabColorKit = [
  FontColorPlugin.withComponent(CollabColorLeaf),
  FontBackgroundColorPlugin.withComponent(CollabColorLeaf),
]
