---
version: alpha
name: Qadam
description: Платформа понятных бизнес-задач и предложений студенческих команд.
colors:
  primary: '#3559e3'
  primary-hover: '#2548cb'
  ink: '#202638'
  muted: '#7b8293'
  line: '#e9ecf2'
  surface: '#ffffff'
  background: '#f7f8fb'
typography:
  body:
    fontFamily: 'Manrope Variable, Manrope, Arial, sans-serif'
  display:
    fontFamily: 'Manrope Variable, Manrope, Arial, sans-serif'
rounded:
  DEFAULT: '16px'
omitted:
  - section: spacing
    reason: Existing responsive geometry remains owned by frontend/src/styles.css.
components:
  button: {}
  card: {}
  dialog: {}
---

# Qadam

## Product character

Qadam is a focused project desk for a business customer and a student team. The visual
language is quiet and professional: white work surfaces, pale blue orientation cues and
cobalt actions. Manrope keeps dense Russian copy readable. Numbered steps only represent
the real task lifecycle: idea, clarification and confirmed card.

The signature interaction is an idea becoming structured evidence. Voice recording uses
the same cobalt system and one restrained red pulse only while the microphone is active.
The Qadam AI emblem is compact; its modal drawer owns the explanatory and chat experience.

## Runtime ownership

`frontend/src/styles.css :root` is the canonical runtime token source. It owns palette,
scrollbar and overlay layers. `--z-sticky`, `--z-dialog` and `--z-toast` define global
stacking. Components must use these variables instead of new arbitrary z-index values.

Desktop uses the fixed 228px sidebar and document scrolling. On mobile the assistant
emblem becomes a floating action and all voice/profile controls wrap without horizontal
overflow. All motion respects `prefers-reduced-motion`.

## Content and identity

The shell is Qadam. AI Sana may appear only as a task owner in seed content. Demo roles
are named directly as «Бизнес» and «Студенческая команда». Email/password language does
not appear in the hackathon MVP. AI output and voice transcripts remain editable and are
never presented as confirmed facts before the user reviews them.
