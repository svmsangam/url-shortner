import { useEffect, useState } from 'react'
import { ArrowUpRight, BarChart3, Camera, Check, Clipboard, Link2, Send } from 'lucide-react'
import api from './api/client'

const cardClass = 'rounded-xl border border-slate-200 bg-white p-6 shadow-sm'

function UrlForm({ longUrl, setLongUrl, shortUrl, shortCode, isSubmitting, isCopied, error, clickCount, isClickCountLoading, clickCountError, onSubmit, onCopy, onViewClicks, onCreateAnother }) {
  return (
    <section className={cardClass} aria-labelledby="create-link-title">
      <div className="mb-6">
        <p className="mb-2 text-xs font-bold uppercase tracking-[0.16em] text-slate-500">Create a link</p>
        <h2 id="create-link-title" className="text-xl font-bold text-slate-800">Shorten your URL</h2>
        <p className="mt-2 text-sm leading-6 text-slate-500">Make a long URL easier to share, remember, and track.</p>
      </div>
      <form onSubmit={onSubmit}>
        <label className="mb-2 block text-sm font-semibold text-slate-700" htmlFor="long-url">Long URL</label>
        <input className="w-full rounded-lg border border-slate-300 bg-slate-50 px-4 py-3 text-sm text-slate-800 outline-none transition placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-100" id="long-url" type="url" value={longUrl} onChange={(event) => setLongUrl(event.target.value)} placeholder="Paste your long url here" required />
        <button className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[#2563EB] px-4 py-3 text-sm font-bold text-white transition hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60" type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Shortening...' : 'Shorten'} <ArrowUpRight size={17} />
        </button>
      </form>
      {error && <p className="mt-4 text-sm text-red-600" role="alert">{error}</p>}
      {shortUrl && <div className="mt-8 border-t border-slate-200 pt-6">
        <label className="mb-2 block text-sm font-semibold text-slate-700" htmlFor="short-url">Shortened URL</label>
        <input className="w-full rounded-lg border border-slate-300 bg-slate-100 px-4 py-3 font-mono text-sm text-slate-700 outline-none" id="short-url" type="text" value={shortUrl} readOnly />
        <p className="mt-2 font-mono text-xs text-slate-400">/{shortCode}</p>
        <div className="mt-5 grid gap-3 sm:grid-cols-2">
          <button className="rounded-lg bg-[#1E293B] px-4 py-3 text-sm font-bold text-white transition hover:bg-slate-700 disabled:cursor-wait disabled:opacity-60" type="button" onClick={onViewClicks} disabled={isClickCountLoading}>{isClickCountLoading ? 'Loading...' : 'View Total Clicks'}</button>
          <button className="rounded-lg bg-[#1E293B] px-4 py-3 text-sm font-bold text-white transition hover:bg-slate-700" type="button" onClick={onCreateAnother}>Create Another</button>
        </div>
        {clickCountError && <p className="mt-4 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600" role="alert">{clickCountError}</p>}
        {clickCount !== null && <p className="mt-4 rounded-lg bg-slate-50 px-3 py-2 text-xs text-slate-500">Total Clicks: {clickCount}</p>}
        <div className="mt-6 flex items-center gap-3 border-t border-slate-100 pt-5"><span className="mr-1 text-sm font-semibold text-slate-500">Share</span><a className="grid h-9 w-9 place-items-center rounded-full bg-blue-600 font-serif text-lg font-bold text-white" href={`https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(shortUrl)}`} target="_blank" rel="noreferrer" aria-label="Share on Facebook" title="Share on Facebook">f</a><a className="grid h-9 w-9 place-items-center rounded-full bg-pink-600 text-white" href="https://www.instagram.com/" target="_blank" rel="noreferrer" aria-label="Open Instagram" title="Open Instagram"><Camera size={16} /></a><a className="grid h-9 w-9 place-items-center rounded-full bg-emerald-600 text-white" href={`https://wa.me/?text=${encodeURIComponent(`Check out this link: ${shortUrl}`)}`} target="_blank" rel="noreferrer" aria-label="Share on WhatsApp" title="Share on WhatsApp"><Send size={16} /></a><button className="ml-auto inline-flex items-center gap-1.5 text-sm font-semibold text-slate-500 hover:text-slate-900" type="button" onClick={onCopy}>{isCopied ? <Check size={16} /> : <Clipboard size={16} />}{isCopied ? 'Copied' : 'Copy'}</button></div>
      </div>}
    </section>
  )
}

function RecentResults({ refreshKey }) {
  const [recentLinks, setRecentLinks] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [copiedCode, setCopiedCode] = useState('')
  const [viewingCode, setViewingCode] = useState('')
  const [clickCounts, setClickCounts] = useState({})
  const [clickCountErrors, setClickCountErrors] = useState({})
  const [loadingClickCode, setLoadingClickCode] = useState('')

  useEffect(() => {
    let isCurrent = true

    async function loadRecentLinks() {
      setIsLoading(true)
      setLoadError('')
      try {
        const { data } = await api.get('/urls')
        if (isCurrent) setRecentLinks(data.records || [])
      } catch {
        if (isCurrent) setLoadError('Recent links could not be loaded.')
      } finally {
        if (isCurrent) setIsLoading(false)
      }
    }

    loadRecentLinks()
    return () => { isCurrent = false }
  }, [refreshKey])

  async function copyLink(shortCode) {
    const shortUrl = `${api.defaults.baseURL.replace(/\/api\/?$/, '')}/${shortCode}`
    try {
      await navigator.clipboard.writeText(shortUrl)
      setCopiedCode(shortCode)
      window.setTimeout(() => setCopiedCode(''), 1800)
    } catch {
      setLoadError('Clipboard access was blocked.')
    }
  }

async function fetchClickCount(shortCode) {
  setViewingCode(shortCode)
  setLoadingClickCode(shortCode)
  try {
    const { data } = await api.get('/v1/urls/clicks/' + shortCode)
    setClickCounts((counts) => ({ ...counts, [shortCode]: data.clicks }))
    setClickCountErrors((errors) => ({ ...errors, [shortCode]: '' }))
  } catch (error) {
    let message = 'Click count could not be loaded.'

    if (error.response) {
      // Server responded with a specific HTTP status code
      switch (error.response.status) {
        case 404:
          message = 'Short code not found.'
          break
        case 401:
        case 403:
          message = 'Unauthorized to view analytics.'
          break
        case 400:
          message = 'Invalid short code request.'
          break
        case 500:
          message = 'Server error. Please try again later.'
          break
        default:
          message = error.response.data?.error || message
      }
    } else if (error.request) {
      // Request was made but no response was received (Network error)
      message = 'Network error. Please check your connection.'
    }

    setClickCountErrors((errors) => ({ ...errors, [shortCode]: message }))
  } finally {
    setLoadingClickCode((code) => (code === shortCode ? '' : code))
  }
}

  return (
    <section className={cardClass} aria-labelledby="recent-results-title">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.16em] text-slate-500">Your links</p>
          <h2 id="recent-results-title" className="text-xl font-bold text-slate-800">Recent Results</h2>
        </div>
        <BarChart3 className="text-slate-400" size={21} aria-hidden="true" />
      </div>
      {isLoading ? <p className="py-12 text-center text-sm text-slate-400">Loading recent links...</p> : loadError ? <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600" role="alert">{loadError}</p> : recentLinks.length ? <div className="space-y-3">{recentLinks.map((link) => {
        const shortUrl = `${api.defaults.baseURL.replace(/\/api\/?$/, '')}/${link.short_code}`
        return <article className="rounded-lg border border-slate-200 bg-slate-50 p-3" key={link.short_code}><div className="flex items-center justify-between gap-3"><a className="min-w-0 truncate font-mono text-sm font-medium text-blue-700 hover:underline" href={shortUrl} target="_blank" rel="noreferrer">{shortUrl}</a><div className="flex shrink-0 gap-2"><button className="rounded-md bg-slate-200 px-2.5 py-1.5 text-xs font-bold text-slate-700 transition hover:bg-slate-300" type="button" onClick={() => copyLink(link.short_code)}>{copiedCode === link.short_code ? 'Copied' : 'Copy'}</button><button className="rounded-md bg-[#1E293B] px-2.5 py-1.5 text-xs font-bold text-white transition hover:bg-slate-700 disabled:cursor-wait disabled:opacity-60" type="button" onClick={() => fetchClickCount(link.short_code)} disabled={loadingClickCode === link.short_code}>{loadingClickCode === link.short_code ? 'Loading...' : 'View Clicks'}</button></div></div>{viewingCode === link.short_code && loadingClickCode !== link.short_code && clickCountErrors[link.short_code] && <p className="mt-2 text-xs text-red-600" role="alert">{clickCountErrors[link.short_code]}</p>}{viewingCode === link.short_code && loadingClickCode !== link.short_code && clickCounts[link.short_code] !== undefined && !clickCountErrors[link.short_code] && <p className="mt-2 text-xs text-slate-500">Total Clicks: {clickCounts[link.short_code]}</p>}</article>
      })}</div> : <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-slate-200 bg-slate-50 px-6 text-center"><Link2 className="mb-3 text-slate-300" size={28} /><p className="text-sm font-semibold text-slate-500">No links yet</p><p className="mt-1 text-xs text-slate-400">Your shortened links will appear here.</p></div>}
    </section>
  )
}

function About() {
  return <footer className="mt-10 border-t border-slate-200 pt-6 text-center text-sm text-slate-500"><h2 className="text-sm font-bold text-slate-700">About</h2><p className="mx-auto mt-2 max-w-xl leading-6">A fast, stateless Go and Cassandra URL shortener built for simple, reliable sharing. No account required.</p></footer>
}

function App() {
  const [longUrl, setLongUrl] = useState('')
  const [shortUrl, setShortUrl] = useState('')
  const [shortCode, setShortCode] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isCopied, setIsCopied] = useState(false)
  const [error, setError] = useState('')
  const [clickCount, setClickCount] = useState(null)
  const [isClickCountLoading, setIsClickCountLoading] = useState(false)
  const [clickCountError, setClickCountError] = useState('')
  const [refreshKey, setRefreshKey] = useState(0)

  async function handleSubmit(event) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)

    try {
      const { data } = await api.post('/shorten', { long_url: longUrl.trim() })
      setShortUrl(data.short_url)
      setShortCode(data.short_code)
      setClickCount(null)
      setClickCountError('')
      setRefreshKey((key) => key + 1)
    } catch (requestError) {
      setError(requestError.response?.data || 'We could not shorten that URL. Try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(shortUrl)
      setIsCopied(true)
      window.setTimeout(() => setIsCopied(false), 1800)
    } catch {
      setError('Clipboard access was blocked. Copy the URL manually instead.')
    }
  }

  async function handleViewClicks() {
    setClickCountError('')
    setIsClickCountLoading(true)
    try {
      const { data } = await api.get('/v1/urls/clicks/' + shortCode)
      setClickCount(data.clicks)
    } catch (error) {
    let message = 'Click count could not be loaded.'

    if (error.response) {
      // Server responded with a specific HTTP status code
      switch (error.response.status) {
        case 404:
          message = 'Short code not found.'
          break
        case 401:
        case 403:
          message = 'Unauthorized to view analytics.'
          break
        case 400:
          message = 'Invalid short code request.'
          break
        case 500:
          message = 'Server error. Please try again later.'
          break
        default:
          message = error.response.data?.error || message
      }
    } else if (error.request) {
      // Request was made but no response was received (Network error)
      message = 'Network error. Please check your connection.'
    }
    setClickCountError(message)
    } finally {
      setIsClickCountLoading(false)
    }
  }

  function createAnother() {
    setLongUrl('')
    setShortUrl('')
    setShortCode('')
    setError('')
    setIsCopied(false)
    setClickCount(null)
    setClickCountError('')
  }

  return <main className="min-h-screen bg-[#F8FAFC] px-4 py-10 text-[#1E293B] sm:px-6"><div className="mx-auto flex min-h-[calc(100vh-5rem)] max-w-5xl flex-col"><h1 className="mb-10 text-center text-3xl font-extrabold tracking-tight text-[#1E293B] sm:text-4xl">URL Shortener</h1><div className="grid flex-1 items-start gap-6 lg:grid-cols-2"><UrlForm longUrl={longUrl} setLongUrl={setLongUrl} shortUrl={shortUrl} shortCode={shortCode} isSubmitting={isSubmitting} isCopied={isCopied} error={error} clickCount={clickCount} isClickCountLoading={isClickCountLoading} clickCountError={clickCountError} onSubmit={handleSubmit} onCopy={handleCopy} onViewClicks={handleViewClicks} onCreateAnother={createAnother} /><RecentResults refreshKey={refreshKey} /></div><About /></div></main>
}

export default App
