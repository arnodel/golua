do
    print("1" + "2", 1 + "3", "2.4" + 2)
    --> =3	4	4.4

    print(pcall(function() return "a" + "1" end))
    --> ~false\t.*attempt to perform arithmetic on a string value

    print(pcall(getmetatable("12").__add))
    --> ~false\t.*2 arguments needed
end
