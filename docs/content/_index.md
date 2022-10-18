---
title: "Pushup web framework"
date: 2022-10-10T16:54:01-05:00
draft: false
markup: html
---
<section>
    <h1><img style="height: 0.75em" src="{{< siteurl >}}logo.png" alt="Pushup logo, a caret surrounded by left and right angle brackets ala HTML element"> Pushup web framework</h1>

    <p>A server-side, page-oriented web framework for the Go programming language.</p>

    <p>Pushup’s goal is to make it faster to develop and easier to maintain server-side web applications using Go.</p>

    <p>Pushup is <b>preview</b>, pre-release software in early-stage development. It is not yet suitable for production use. Expect breaking changes.</p>
</section>

<section id="code-example">
    <h2>Pushup example</h2>
    <pre><code><span class="keyword">^import</span> <span class="go">"time"</span>

<span class="keyword">^</span>{
   <span class="go">title := "Hello, from Pushup!"</span>
}

<span class="html">&lt;h1&gt;</span><span class="simple-expr">^title</span><span class="html">&lt;/h1&gt;</span>

<span class="html">&lt;p&gt;</span>The time is now <span class="simple-expr">^time.Now().String()</span>.<span class="html">&lt;/p&gt;</span>

<span class="keyword">^if</span> <span class="go">time.Now().Weekday() == time.Friday</span> {
    <span class="html">&lt;p&gt;</span>It's Friday! Enjoy the start to your weekend.<span class="html">&lt;/p&gt;</span>
} <span class="keyword">^else</span> {
    <span class="html">&lt;p&gt;</span>Have a great day, we're glad you're here.<span class="html">&lt;/p&gt;</span>
}
</code></pre>
    <p>File <kbd>app/pages/hello.up</kbd> → <kbd>/hello</kbd> URL path</p>
</section>

<section>
    <h2>Features</h2>

    <ul id="feature-list">
        <li>
            <strong>Page-oriented</strong>
            <p>Pushup’s .up files are self-contained units of web app development, gluing HTML &amp; Go together with routing logic</p>
        </li>
        <li>
            <strong>File-based routing</strong>
            <p>Filesystem path names of Pushup pages map to URL paths, with dynamic escape hatches</p>
        </li>
        <li>
            <strong>World’s first ‟<a href="https://htmx.org/">htmx</a>-first” framework</strong>
            <p>Enhanced hypertext support via inline partials for better client-side interactivity with fewer JavaScript sit-ups</p>
        </li>
        <li>
            <strong>Compiled</strong>
            <p>Pushup apps compile to pure Go, built on the standard <code>net/http</code> package. Fast static binary executables for easy deployment. Easy to integrate into larger Go apps</p>
        </li>
        <li>
            <strong>Hot reload dev mode</strong>
            <p>App is recompiled and reloaded in the browser while files change during development. This is fast thanks to the Go compiler</p>
        </li>
    </ul>
</section>

<section>
    <h2>Getting started</h2>

    <ul>
        <li><strong>Download Pushup</strong>
            <p>Official release TBD. For now, grab and build from <a href="https://github.com/adhocteam/pushup">git</a>.</p>
        </li>
        <li><strong>Read the documentation</strong>
            <p><a href="{{< siteurl >}}docs/">Pushup docs</a></p>
        </li>
        <li><strong>Read the source &amp; join the community</strong>
            <p><a href="https://github.com/adhocteam/pushup">GitHub repo</a></p>
        </li>
    </ul>
</section>
