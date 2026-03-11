# Design Doc: Replace Activities Page Date Input with Shadcn Date Range Picker

## Overview
The current Activities page uses two separate `datetime-local` input fields for selecting the start and end dates of the operation logs. This design aims to replace them with a single, intuitive `DatePickerWithRange` component from shadcn/ui.

## Goals
- Replace legacy `datetime-local` inputs with a modern date range picker.
- Improve user experience by providing a visual calendar for range selection.
- Maintain compatibility with the existing API filtering logic (`start` and `end` ISO strings).
- Set a sensible default range (last 7 days).

## Architecture & Components

### Component Structure
- **Trigger**: A `Button` within a `PopoverTrigger` that displays the selected range (or a placeholder if empty).
- **Content**: A `PopoverContent` containing the `Calendar` component in `range` mode.
- **Calendar**: Configured with `numberOfMonths={2}` to allow easier cross-month range selection.

### State Management
- **Internal State**: A `DateRange` object (`{ from: Date, to: Date }`) to manage the calendar selection.
- **Sync Logic**: 
  - On `DateRange` change: Update the existing `start` and `end` string states in the `ActivitiesPage` (formatted as ISO strings or empty).
  - On Page Load: Initialize the `DateRange` from the `start` and `end` states (if they exist).

### Default State
- The `ActivitiesPage` will initialize `start` and `end` to represent the last 7 days (e.g., using `subDays(new Date(), 7)` and `new Date()`).

## Implementation Plan

### 1. Component Refactoring
- Add `DatePickerWithRange` to `ActivitiesPage`.
- Move the state management for the date range into the `ActivitiesPage` or a dedicated sub-component.

### 2. Integration
- Replace the two `<Input type="datetime-local" />` components in `toolbarLeft` with the new range picker.
- Ensure the `onValueChange` correctly updates the `pagination` (resetting to page 0) to trigger a data re-fetch.

### 3. Styling
- Use the existing `Calendar` and `Popover` components in `ui/src/components/ui`.
- Ensure the trigger button matches the style of other toolbar elements (like the `Combobox`).

## Verification Plan
- **UI Verification**: Ensure the calendar opens correctly and the range is visually highlighted.
- **Functional Verification**: Check if selecting a range updates the table data correctly.
- **Edge Cases**: Verify behavior when only a single date is selected (range is incomplete).
- **Diagnostics**: Run `lsp_diagnostics` on `activities-page.tsx`.
