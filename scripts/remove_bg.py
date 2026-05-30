from PIL import Image

def remove_black_background(img_path):
    img = Image.open(img_path).convert("RGBA")
    datas = img.getdata()

    newData = []
    for item in datas:
        # Check if the pixel is black or very dark
        # Threshold can be adjusted. (20,20,20) covers almost black.
        if item[0] < 30 and item[1] < 30 and item[2] < 30:
            # Change to transparent
            newData.append((255, 255, 255, 0))
        else:
            newData.append(item)

    img.putdata(newData)
    img.save(img_path, "PNG")
    print("Processed image and removed black background.")

remove_black_background(r"C:\projects\handayani-com\public\LogoHandayani.png")
