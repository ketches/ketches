'use client'

import { FontBackgroundColorPlugin, FontColorPlugin } from "@platejs/basic-styles/react"

import { CollabColorLeaf } from "../collab-editor-elements"

export const CollabColorKit = [
  FontColorPlugin.withComponent(CollabColorLeaf),
  FontBackgroundColorPlugin.withComponent(CollabColorLeaf),
]
