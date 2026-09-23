export default function Notice({ text, tone = 'info' }: { text: string; tone?: 'info' | 'error' }) {
  if (!text) return null
  return <p className={`message ${tone}`}>{text}</p>
}
