import sys
import json
import os

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8')
else:
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

def parse_pdf_fallback(pdf_path):
    try:
        import pypdf
        reader = pypdf.PdfReader(pdf_path)
        full_text = ""
        for page in reader.pages:
            text = page.extract_text()
            if text:
                full_text += text + "\n"
        return {
            "name": os.path.basename(pdf_path),
            "text": full_text,
            "tables": [],
            "sections": [],
            "pages": len(reader.pages),
            "metadata": {}
        }
    except Exception as e:
        return {"error": str(e)}

def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No PDF path"}), file=sys.stderr)
        sys.exit(1)
    
    pdf_path = sys.argv[1]
    if not os.path.exists(pdf_path):
        print(json.dumps({"error": f"File not found: {pdf_path}"}), file=sys.stderr)
        sys.exit(1)
    
    result = parse_pdf_fallback(pdf_path)
    
    if "error" in result:
        print(json.dumps(result), file=sys.stderr)
        sys.exit(1)
    
    output = json.dumps(result, ensure_ascii=False)  # заменяю проблемные символы
    output = output.replace('\u2212', '-')
    print(output)

if __name__ == "__main__":
    main()