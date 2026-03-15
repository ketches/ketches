'use client'

import { CollabBlocksKit } from "./collab-blocks-kit"
import { CollabColorKit } from "./collab-color-kit"
import { CollabAutoformatKit } from "./collab-autoformat-kit"
import { CollabLinkKit } from "./collab-link-kit"
import { CollabListsKit } from "./collab-lists-kit"
import { CollabMarksKit } from "./collab-marks-kit"

export const CollabEditorKit = [
  ...CollabBlocksKit,
  ...CollabMarksKit,
  ...CollabColorKit,
  ...CollabListsKit,
  ...CollabLinkKit,
  CollabAutoformatKit,
]
