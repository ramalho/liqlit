#!/usr/bin/env python
import os
import sys
import re

try:
    import pymupdf4llm
except ImportError:
    print("Error: pymupdf4llm is not installed. Please run: pip install pymupdf4llm")
    sys.exit(1)

def clean_markdown(text):
    """
    Cleans up common artifacts from PDF text extraction.
    """
    # 1. Fix hyphens split across lines (e.g., al-\nienista -> alienista)
    text = re.sub(r'(\w+)-\n\s*(\w+)', r'\1\2', text)
    
    # 2. Join lines that break inside sentences, keeping double newlines for paragraphs
    lines = text.split('\n')
    cleaned_lines = []
    buffer = ""
    
    for line in lines:
        stripped = line.strip()
        if not stripped:
            if buffer:
                cleaned_lines.append(buffer)
                buffer = ""
            cleaned_lines.append("") # Keep paragraph spacing
            continue
            
        # If line looks like a markdown header, list, or separator, flush buffer first
        if stripped.startswith('#') or stripped.startswith('-') or stripped.startswith('*') or re.match(r'^\d+\.', stripped):
            if buffer:
                cleaned_lines.append(buffer)
                buffer = ""
            cleaned_lines.append(line)
        else:
            if buffer:
                # If buffer ends with punctuation, maybe it's the end of a sentence
                if buffer[-1] in ['.', '!', '?']:
                    cleaned_lines.append(buffer)
                    buffer = line
                else:
                    buffer += " " + line
            else:
                buffer = line
                
    if buffer:
        cleaned_lines.append(buffer)
        
    return '\n'.join(cleaned_lines)

def convert_pdf_to_md(pdf_path):
    if not os.path.exists(pdf_path):
        print(f"Error: File '{pdf_path}' not found.")
        sys.exit(1)
        
    base_name = os.path.splitext(pdf_path)[0]
    output_path = f"{base_name}.md"
    
    print(f"Extracting markdown from {pdf_path} using pymupdf4llm...")
    try:
        # Extract markdown using pymupdf4llm
        md_text = pymupdf4llm.to_markdown(pdf_path)
        
        print("Cleaning up text artifacts and formatting...")
        clean_text = clean_markdown(md_text)
        
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(clean_text)
            
        print(f"Success! Saved clean markdown to: {output_path}")
        
    except Exception as e:
        print(f"An error occurred during conversion: {e}")

if __name__ == "__main__":
    if len(sys.argv) > 1:
        input_pdf = sys.argv[1]
    else:
        input_pdf = "o-alienista.pdf"
        
    convert_pdf_to_md(input_pdf)
