# OAuth Flow Guide

Simple OAuth 2.0 flow to authenticate and access documents.

## Flow Steps

### 1. Request Authorization

```bash
curl --location 'http://localhost:8080/api/v1/oauth/authorize' \
--header 'Content-Type: application/json' \
--data '{}'
```

### 2. Get Token Response

After hitting the authorization URL, you'll get a response like:

```json
{
  "token_type": "Bearer",
  "scope": "User.Read Files.Read Notes.Read",
  "expires_in": 3599,
  "ext_expires_in": 3599,
  "access_token": "EwBYBMl6BAAUBKgm8k1UswUNwklmy2v7U/S+1fEAAUupkM+QlFc1vHbononKn3eSn9lv+2Ftm5a8W3xBAmp0fUgckdWVHRN9lR4aedUppQ6wfh0LD+S6OE3UAO8Xq4Dp/IYnt9px09ZqQJ0EXlfwr5Lgi3lGlwsszlGDm94KbkMpUBaDSMpQ2R751LV68YF9Wr7XVxHRFN5Ml59dbSO/dXmj6kRB2yUtuwdSgfsK40R8Ea4HedNh7LUEosxQxwjM8NG6iT6nQYL4dplUaC7M8whe8oyuCSdCypVZ/iA0vm4CZPlnWywke0RoL8UpdCECj88IbIR2GC+aSwn23rxzM0Q0vD7E9EXk2Ac6NowYbr+rJmbsnC0UIVWLu1nq4rkQZgAAEBCHWyHYo0kQC5tymExVTzAgA2hb6/dGiL6h1WzUGDORTymu7/Q8SuoOVUG00kFNAyTHsy/VpO4MRBPZBDAtBIu2XAHj2E1GP+yt7HebE1a2QE5r62siSmuSDTwPWz0VISlBqLfLDLlm1DCoDGxKEQ4TL2eIAebO3/71xyBtYCv5tdKlrx+lEw5hhZLEb2PDDwGSgsF+GmWqjUbW5JHhZ+gQpFwu6eQKOfeFD9oPO5ySf4R+tzTVOkvxZGHPHhIUhTcGtdRHX7ak+uRhSsEBY+RGuIzF9SwRunmevxYRS7E+6s3Tk87rohFjpFPlSyptYiPl0Ti3puGUk4c6AHEXp2rdXjzFXRqTW7Z/N0QoeIyCedcZK4xSZNE/S0Gbr0ASNvEOKo52pYn+jZ95fQcRLtq+lfJO97EXuPMu3K6auSGaxAY5HROSoQKg7T1sM0nZXRLMdvVySiz0GtnyIvbxxYRIpsrFZDlo5fP7R5Z+W1YcLvaWFJ88a4gLxhcURQ1i+Pm4u/RO5TPK+7TFcYpoe8cGhQlYKV8EoZ21oNhMjOdnCiux/z4rAPM39Jfh1cSeLm4fB4PbXsJeQN97L0ToKqKTqC9KR+3Y/8cP5ikS6vhHkZH+3JVIcLaen5xvBG4hz6KKMUlkTVBUaF2xeWNWiWI9oSztVZQAV+tbg4p+a1OKD7pAV5Uhh1RPkuvy6niib8Zeog5Jd7wYazaVLSxBmVHXmD2snd7+OYxC70Xe67mBpj5lpr9InEHRWITbpFqEGb4+nFecihvKyv5yeN0MgdydUbweN6jQ+l4uPjsh5ZK3ColywySCFeI/qu0T5fNsxQ9prJKNqIEXRywZlF1IS0/w/3UT3VmwlOJSM/RSUziqTU4dBqpD++NYwm8RxwolpE5dopd6RgSqj9AAg+ADYTNbdmdZc5Hxc5Akd/Rbg2b3fNl1ntYlDxdP4mzGtriZZm/eNO88nX34Ouep4oXnbaSSYBN3JVu/7Qs+Ku3EQuBFEhKAYKVr4L5s/OjRAovTN2OQdAHU1QIDcspDYVAIp1v+BC5fZXvZHgkUOUm9gRJvZ/qUvcJSE/1qpBYajYF3Ek3vZwM=",
  "refresh_token": "M.C544_BAY.0.U.-Cj0Spb1ZxAAk1m7At5WYrdEkHnvTPA0LbbOmRBDbpsZeD6P!4ZVqUKQR2Krugd6WTtRXFAHsl5FCGeeZzX7EIxZcIa1nvJITqpJq!AfLGnrCD1orw3vLcleuYFwOgw6QMPWStejxRs0c0jIUfOUj*jxr7XFFBdFb2sJDE4Wr3fJJRvglLOMvdLPr7XvWIGFHMdrwDHci*RhjxZXyD7mBXVySP40SP5iLm0PelmPO7epa28Cpz0R5BbTbyb8V6nUFaFf4PliVvMtpzJeGt9HdgfiUETVwzG!77DFjZed92o!3Qte2uSTLSAaoZPUXL3lj33ELoqD447TQiLtya1HraDvI*CyQilE4Sa7*3I1UbIOrdC9Np3THIO7gReL83HGWhnSrcLEDQwrkRBRVpG6r*Z76Hf7fBC1ULjX9!rB15K0m"
}
```

### 3. Use Access Token to Extract Data

From the response above, take the `access_token` and call the `/pipeline` API:

```bash
curl --location 'http://localhost:8080/api/v1/pipeline' \
--header 'Authorization: Bearer YOUR_ACCESS_TOKEN_HERE'
```

That's it! You now have access to extract data through the OAuth flow. 