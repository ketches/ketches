'use client'

import type { Value } from "platejs"

import { Plate, usePlateEditor } from "platejs/react"
import { useEffect, useRef, useState } from "react"

import { Editor, EditorContainer } from "@/components/ui/editor"
import { cn } from "@/lib/utils"

import { CollabEditorToolbar } from "./collab-editor-toolbar"
import {
  deserializeCollabEditorValue,
  serializeCollabEditorValue,
} from "./collab-editor-value"
import { CollabEditorKit } from "./plugins/collab-editor-kit"

interface CollabEditorProps {
  className?: string
  onChange?: (value: string) => void
  placeholder?: string
  value?: string
}

export function CollabEditor({
  className,
  onChange,
  placeholder = "Type your amazing content here...",
  value,
}: CollabEditorProps) {
  const [initialValue] = useState<Value>(() => deserializeCollabEditorValue(value))
  const lastSerializedValueRef = useRef(serializeCollabEditorValue(initialValue))

  const editor = usePlateEditor({
    plugins: CollabEditorKit,
    value: initialValue,
  })

  useEffect(() => {
    const nextValue = deserializeCollabEditorValue(value)
    const nextSerializedValue = serializeCollabEditorValue(nextValue)

    if (nextSerializedValue === lastSerializedValueRef.current) {
      return
    }

    editor.tf?.setValue?.(nextValue)
    lastSerializedValueRef.current = nextSerializedValue
  }, [editor, value])

  return (
    <div className={cn("overflow-hidden rounded-md border bg-background", className)}>
      <Plate
        editor={editor}
        onValueChange={({ value: currentValue }) => {
          const serializedValue = serializeCollabEditorValue(currentValue)

          lastSerializedValueRef.current = serializedValue
          onChange?.(serializedValue)
        }}
      >
        <CollabEditorToolbar editor={editor} />
        <EditorContainer variant="select" className="border-none">
          <Editor className="min-h-32" placeholder={placeholder} variant="select" />
        </EditorContainer>
      </Plate>
    </div>
  )
}
