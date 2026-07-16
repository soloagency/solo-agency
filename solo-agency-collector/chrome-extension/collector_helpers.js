/*
 * collector_helpers.js
 * Shared DOM/text helpers for the Solo Agency Local Collector.
 *
 * These functions are lifted verbatim from gubo-remotion-player/js/autobots.js
 * so that readability.js and infinity_loops.js can run self-contained inside the
 * extension's injected (isolated-world) context. Load this file BEFORE
 * readability.js and infinity_loops.js.
 *
 * The canonical link-annotation convention lives in getHumanReadableText():
 * every <a> becomes `text(url)` (with mailto:/tel: special-cased) so the LLM
 * downstream keeps the URL attached to the text it came from.
 */

function extractPlainText_and_a_tag(htmlString) {
  const parser = new DOMParser();
  const doc = parser.parseFromString(htmlString, "text/html");
  const tempElement = doc.body; // The DOMParser has already cleaned up the HTML

  // Remove script, style, and noscript tags
  ['script', 'style', 'noscript', 'a'].forEach(tagName => {
    const elements = tempElement.getElementsByTagName(tagName);
    for (let i = elements.length - 1; i >= 0; i--) {
      elements[i].remove();
    }
  });

  let text = tempElement.textContent || tempElement.innerText || '';

  // Remove emails and phone numbers
  text = text.replace(/[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g, ''); // Remove email addresses
  text = text.replace(/(?:\+\d{1,3})?\d{10,14}/g, ''); // Remove phone numbers

  return text;
}

function extractPlainText(htmlString) {
  const parser = new DOMParser();
  const doc = parser.parseFromString(htmlString, "text/html");
  const tempElement = doc.body; // The DOMParser has already cleaned up the HTML

  // Remove script, style, and noscript tags
  ['script', 'style', 'noscript'].forEach(tagName => {
    const elements = tempElement.getElementsByTagName(tagName);
    for (let i = elements.length - 1; i >= 0; i--) {
      elements[i].remove();
    }
  });

  return tempElement.textContent || tempElement.innerText || '';
}

function cleanText(text) {
    // Replace sequences of spaces with a single space.
    text = text.replace(/ +/g, ' ');
    // Replace sequences of line breaks with a single line break.
    text = text.replace(/\n+/g, '\n');
    // Replace space followed by a line break with a line break.
    text = text.replace(/ \n/g, '\n');
    // Replace line break followed by a space with a line break.
    text = text.replace(/\n /g, '\n');

    text = text.replace(/\t+/g, '\n');
    // Replace space followed by a line break with a line break.
    text = text.replace(/ \t/g, '\n');
    // Replace line break followed by a space with a line break.
    text = text.replace(/\t /g, '\n');
    // Trim leading and trailing whitespace
    text = text.replace(/\n+/g, '\n')
    text = text.trim();
    return text;
}

function getHumanReadableText(htmlString, CTA_Titles) {
    CTA_Titles = CTA_Titles || [];
    const parser = new DOMParser();
    const doc = parser.parseFromString(htmlString, "text/html");

    // Convert matching input or buttons to divs with the text
    CTA_Titles.forEach(title => {
        const titleLowercase = title.toLowerCase();
        doc.querySelectorAll('input, button').forEach(node => {
            const attributes = Array.from(node.attributes).map(attr => attr.value.toLowerCase());
            if (attributes.includes(titleLowercase) || (node.innerText && node.innerText.toLowerCase() === titleLowercase)) {
                const div = doc.createElement('div');
                div.textContent = title;
                node.replaceWith(div);
            }
        });
    });

    // Handle <a> tags
    doc.querySelectorAll('a').forEach(a => {
        let linkText;
        if (a.href && !a.href.startsWith('javascript:') && !a.href.startsWith('#') && !a.href.startsWith('void(0)')) {
            if (a.href.startsWith('mailto:')) {
                linkText = `Email: ${a.textContent}(${a.href.replace('mailto:', '')})`;
            } else if (a.href.startsWith('tel:')) {
                linkText = `Phone: ${a.textContent}(${a.href.replace('tel:', '')})`;
            } else {
                linkText = `${a.textContent}(${a.href})`;
            }
        } else {
            // Keeping text only for special href cases
            linkText = a.textContent;
        }
        const textNode = doc.createTextNode(linkText);
        a.replaceWith(textNode);
    });

    // Remove script tags from the document
    const nonContentNodes = doc.querySelectorAll('script, style, noscript, iframe');
    nonContentNodes.forEach(node => node.remove());

    // Insert spaces before and after each text node to ensure spaces between elements and their text
    doc.body.querySelectorAll('*').forEach(el => {
        Array.from(el.childNodes).forEach(child => {
            if (child.nodeType === 3) { // Check if it's a text node
                el.insertBefore(doc.createTextNode(' '), child);
                el.insertBefore(doc.createTextNode(' '), child.nextSibling);
            }
        });
    });

    // Extract text and remove placeholders
    let text = doc.body.innerText.trim();

    text = text.replace(/%.*?%/g, '');  // remove %...% placeholders
    text = text.replace(/\$\{.*?\}/g, ''); // remove ${...} placeholders
    text = text.replace(/\s+/g, ' '); // condense multiple spaces into a single space
    return text;
}

function extractRootDomain(url) {
    let domain;
    try {
        domain = new URL(url).hostname;
    } catch (error) {
        console.error("Invalid URL:", url);
        return null;  // or return an empty string if you prefer: return "";
    }

    let parts = domain.split('.').reverse();
    if (parts.length > 2 && !(/^\d+$/.test(parts[0]))) {
        // Exclude IP addresses
        return parts[1] + '.' + parts[0];
    }
    return domain;
}

function extractURLs(htmlCode, mainURL, parent_title) {
  let urlData = [];
  let mainDomain = extractRootDomain(mainURL);

  // Use a DOMParser to convert the HTML string to a document
  let parser = new DOMParser();
  let doc = parser.parseFromString(htmlCode, 'text/html');

  // Get all 'a' elements in the document
  let links = doc.querySelectorAll('a');

  // Iterate over each link
  for (let link of links) {
    // Get the href attribute of the link
    let url = link.getAttribute('href');

    if(url!=null){
      // Get the text content of the link
      let text = link.textContent.trim();

      // If the URL is relative (doesn't start with 'http'), create an absolute URL
      if (!url.startsWith('http')) {
        url = new URL(url, mainURL).toString();
      }

      // Check if the URL's domain matches the main domain or is a subdomain
      let urlDomain = extractRootDomain(url);
      const allowedDomains = ['cdn.shopify.com', '.cloudflare.com', 'akamaized.net', '.fastly.net','edgecastcdn.net','amazonaws.com','akamaihd.net','akamaiedge.net','maxcdn.com','azureedge.net','kxcdn.com','stackpathcdn.com'];

    if (urlDomain && (urlDomain === mainDomain || urlDomain.endsWith('.' + mainDomain) || allowedDomains.some(domain => url.includes(domain)))) {
        // Remove fragment identifier from the URL
        url = url.split('#')[0];
        urlData.push({ url: url, text: text, status:0 });
      }
  }

  }

  // Remove duplicated URLs
  let uniqueUrlData = [];
  let urlStrings = [];
  for (let data of urlData) {
    if (!urlStrings.includes(data.url)) {
      uniqueUrlData.push(data);
      urlStrings.push(data.url);
    }
  }

  // Filter out static file URLs except for PDF and document URLs
  uniqueUrlData = uniqueUrlData.filter(function ({ url }) {
    let extension = url.split('?')[0].split('.').pop().toLowerCase();
    let versioning = url.includes('.css?') || url.includes('.js?');
    return !['css', 'js', 'jpg', 'jpeg', 'png', 'gif'].includes(extension) && !versioning;
  });

  return uniqueUrlData;
}

function findPriceNearCTA(text, CTA_Titles, radius = 200) {
    const pricePatterns = [
        { pattern: /\$\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "$" },
        { pattern: /USD\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "$" },
        { pattern: /\d+(,\d{3})*\s?VND(?!\w)/ig, currency: "VND" },
        { pattern: /€\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "€" },
        { pattern: /EUR\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "€" },
        { pattern: /₩\s?\d+(,\d{3})*(?!\w)/ig, currency: "₩" },
        { pattern: /KRW\s?\d+(,\d{3})*(?!\w)/ig, currency: "₩" },
        { pattern: /¥\s?\d+(,\d{3})*(?!\w)/ig, currency: "¥" },
        { pattern: /JPY\s?\d+(,\d{3})*(?!\w)/ig, currency: "¥" },
        { pattern: /£\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "£" },
        { pattern: /GBP\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "£" },
        { pattern: /A\$\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "A$" },
        { pattern: /AUD\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "A$" },
        { pattern: /C\$\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "C$" },
        { pattern: /CAD\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "C$" },
        { pattern: /\d+(,\d{3})*\s?CHF(?!\w)/ig, currency: "CHF" },
        { pattern: /\d+(,\d{3})*\s?SEK(?!\w)/ig, currency: "SEK" },
        { pattern: /NZ\$\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "NZ$" },
        { pattern: /NZD\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "NZ$" },
        { pattern: /SG\$\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "SG$" },
        { pattern: /SGD\s?\d+(,\d{3})*(\.\s?\d{1,2})?(?!\w)/ig, currency: "SG$" }
    ];

     let cleanedText = text.replace(/\s+\.+\s+|\s+|\n+/g, ' ');

    let allResults = [];
    for (let cta of CTA_Titles) {
        let position = cleanedText.toLowerCase().indexOf(cta.toLowerCase());

        while (position !== -1) {
            const beforeCTA = cleanedText.slice(Math.max(0, position - radius), position);

            let prices = [];
            for (let { pattern } of pricePatterns) {
                let matches = beforeCTA.match(pattern);
                if (matches) {
                    for (let match of matches) {
                        prices.push({
                            price: match.replace(/\s/g, ''),
                            index: beforeCTA.lastIndexOf(match)
                        });
                    }
                }
            }

            if (prices.length) {
                // Sort by the proximity to the CTA, with prices closest to the CTA first
                prices.sort((a, b) => b.index - a.index);

                let nearestPrices = [prices[0].price];

                // Check word count between two closest prices
                if (prices.length > 1) {
                    const segmentBetweenPrices = beforeCTA.slice(prices[1].index + prices[1].price.length, prices[0].index);
                    const wordCount = segmentBetweenPrices.split(/\s+/).length;
                    if (wordCount <= 3 && nearestPrices.indexOf(prices[1].price)==-1) {
                        nearestPrices.push(prices[1].price);
                    }
                }

                if (prices.length > 2 && nearestPrices.length<2) {
                    const segmentBetweenPrices = beforeCTA.slice(prices[2].index + prices[2].price.length, prices[1].index);
                    const wordCount = segmentBetweenPrices.split(/\s+/).length;
                    if (wordCount <= 3 && nearestPrices.indexOf(prices[2].price)==-1) {
                        nearestPrices.push(prices[2].price);
                    }
                }


                allResults.push({
                    cta: cta,
                    prices: nearestPrices,
                    position: position
                });
            }

            position = cleanedText.toLowerCase().indexOf(cta.toLowerCase(), position + 1);
        }
    }

    // If there are results, return the one corresponding to the CTA closest to the beginning of the text.
    if (allResults.length) {
        allResults.sort((a, b) => a.position - b.position);
        return {
            cta: allResults[0].cta,
            prices: allResults[0].prices
        };
    }

    return "";
}
